import React, { useMemo } from 'react';
import { AppHeader, GameCard } from 'activity-hub-sdk';
import GameBoard from './components/GameBoard';

// Parse query params from URL
function useQueryParams() {
  return useMemo(() => {
    const params = new URLSearchParams(window.location.search);
    return {
      gameId: params.get('gameId'),
      userId: params.get('userId'),
      userName: params.get('userName') || 'Player',
      token: params.get('token'),
    };
  }, []);
}

function App() {
  const { gameId, userId, userName, token } = useQueryParams();

  // Must have userId and token to play
  if (!userId || !token) {
    return (
      <>
        <AppHeader title="Bulls and Cows" icon="🐂" />
        <GameCard size="narrow">
          <p className="ah-meta">
            Missing authentication. Please access this game through the Activity Hub.
          </p>
        </GameCard>
      </>
    );
  }

  // Must have gameId to play
  if (!gameId) {
    return (
      <>
        <AppHeader title="Bulls and Cows" icon="🐂" />
        <GameCard size="narrow">
          <p className="ah-meta">
            No game found. Please start a new game from the lobby.
          </p>
        </GameCard>
      </>
    );
  }

  return <GameBoard gameId={gameId} token={token} userId={userId} userName={userName} />;
}

export default App;
