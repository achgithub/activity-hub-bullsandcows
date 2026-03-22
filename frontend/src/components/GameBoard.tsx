import { useState, useEffect, useCallback } from 'react';
import { AppHeader } from 'activity-hub-sdk';
import { useGameSocket, SSEEvent } from '../hooks/useGameSocket';
import CodeSettingPhase from './CodeSettingPhase';
import TwoPlayerBoard from './TwoPlayerBoard';
import SoloPlayerBoard from './SoloPlayerBoard';

interface Guess {
  id: number;
  gameId: string;
  turnNumber: number;
  playerId: string;
  guessCode: string;
  bulls: number;
  cows: number;
  guessedAt: string;
}

interface Game {
  id: string;
  mode: string;
  variant: string;
  maxGuesses: number;
  status: string;
  winner?: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;

  // Solo play fields
  secretCode?: string;
  codeBreaker?: string;

  // 2-player fields
  player1Id?: string;
  player2Id?: string;
  player1Code?: string;
  player2Code?: string;
  player1CodeSet?: boolean;
  player2CodeSet?: boolean;
  currentTurn?: number;

  guesses: Guess[];
}

interface GameBoardProps {
  gameId: string;
  token: string;
  userId: string;
  userName: string;
}

export default function GameBoard({ gameId, token, userId }: GameBoardProps) {
  const [game, setGame] = useState<Game | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchGame = useCallback(async () => {
    try {
      // Use absolute URL with proxy path
      const response = await fetch(`/api/apps/bulls-and-cows/proxy/api/game/${gameId}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-User-ID': userId,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to fetch game');
      }

      const data = await response.json();
      // Ensure guesses array exists
      if (!data.guesses) {
        data.guesses = [];
      }
      setGame(data);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching game:', err);
      setError('Failed to load game');
      setLoading(false);
    }
  }, [gameId, token, userId]);

  // Handle SSE events
  const handleSSEEvent = useCallback((event: SSEEvent) => {
    console.log('SSE event received:', event.type, event.payload);

    if (event.type === 'guess_made') {
      // Solo play guess
      fetchGame();
    } else if (event.type === 'code_set') {
      // One player set code
      fetchGame();
    } else if (event.type === 'both_codes_set') {
      // Both players set codes - game starting
      fetchGame();
    } else if (event.type === 'guess_submitted') {
      // One player submitted guess for current turn
      // No-op: waiting state is managed in TwoPlayerBoard component
    } else if (event.type === 'turn_complete') {
      // Both players submitted - turn evaluated
      fetchGame();
    }
  }, [fetchGame]);

  useGameSocket(gameId, token, handleSSEEvent);

  // Fetch initial game state
  useEffect(() => {
    fetchGame();
  }, [fetchGame]);

  const handleSoloGuess = async (guess: string) => {
    const response = await fetch(`/api/apps/bulls-and-cows/proxy/api/game/${gameId}/guess`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'X-User-ID': userId,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ guess }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(errorText || 'Failed to submit guess');
    }

    // Update game with new guess
    fetchGame();
  };

  const handleTwoPlayerGuess = async (guess: string) => {
    const response = await fetch(`/api/apps/bulls-and-cows/proxy/api/game/${gameId}/guess`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'X-User-ID': userId,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ guess }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(errorText || 'Failed to submit guess');
    }

    const data = await response.json();

    if (data.bothGuessed) {
      // Both players guessed - wait for turn_complete event
      fetchGame();
    }
    // Waiting state is managed in TwoPlayerBoard component
  };

  const renderHeader = () => <AppHeader title="Bulls and Cows" icon="🐂" />;

  if (loading) {
    return (
      <>
        {renderHeader()}
        <div className="ah-container ah-container--narrow">
          <div className="ah-card">
            <p>Loading game...</p>
          </div>
        </div>
      </>
    );
  }

  if (error || !game) {
    return (
      <>
        {renderHeader()}
        <div className="ah-container ah-container--narrow">
          <div className="ah-card">
            <div className="ah-banner ah-banner--error">
              {error || 'Failed to load game'}
            </div>
          </div>
        </div>
      </>
    );
  }

  // Route to appropriate component based on variant and status
  if (game.variant === '1player') {
    // Solo play
    return (
      <>
        {renderHeader()}
        <SoloPlayerBoard
          mode={game.mode}
          secretCode={game.secretCode}
          guesses={game.guesses}
          maxGuesses={game.maxGuesses}
          status={game.status}
          onSubmitGuess={handleSoloGuess}
        />
      </>
    );
  } else if (game.variant === '2player') {
    if (game.status === 'code_setting') {
      // Code setting phase
      const isPlayer1 = game.player1Id === userId;
      const myCodeSet = isPlayer1 ? game.player1CodeSet || false : game.player2CodeSet || false;
      const opponentCodeSet = isPlayer1 ? game.player2CodeSet || false : game.player1CodeSet || false;

      return (
        <>
          {renderHeader()}
          <CodeSettingPhase
            gameId={gameId}
            token={token}
            userId={userId}
            mode={game.mode}
            myCodeSet={myCodeSet}
            opponentCodeSet={opponentCodeSet}
            onCodeSet={fetchGame}
          />
        </>
      );
    } else {
      // Active gameplay or game over
      const isPlayer1 = game.player1Id === userId;
      const myCode = isPlayer1 ? game.player1Code || '' : game.player2Code || '';
      const myGuesses = game.guesses.filter(g => g.playerId === userId);
      const opponentGuesses = game.guesses.filter(g => g.playerId !== userId);
      const opponentLastGuess = opponentGuesses.length > 0 ? opponentGuesses[opponentGuesses.length - 1] : null;

      return (
        <>
          {renderHeader()}
          <TwoPlayerBoard
            userId={userId}
            mode={game.mode}
            myCode={myCode}
            opponentLastGuess={opponentLastGuess}
            myGuesses={myGuesses}
            currentTurn={game.currentTurn || 1}
            maxGuesses={game.maxGuesses}
            status={game.status}
            winner={game.winner}
            onSubmitGuess={handleTwoPlayerGuess}
          />
        </>
      );
    }
  }

  return (
    <>
      {renderHeader()}
      <div className="ah-container ah-container--narrow">
        <div className="ah-card">
          <div className="ah-banner ah-banner--error">
            Unknown game variant: {game.variant}
          </div>
        </div>
      </div>
    </>
  );
}
