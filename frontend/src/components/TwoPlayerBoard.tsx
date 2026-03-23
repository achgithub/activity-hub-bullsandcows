import { useState, useEffect } from 'react';

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

interface TwoPlayerBoardProps {
  userId: string;
  mode: string;
  myCode: string;
  opponentLastGuess: Guess | null;
  myGuesses: Guess[];
  currentTurn: number;
  maxGuesses: number;
  status: string;
  winner?: string;
  onSubmitGuess: (guess: string) => Promise<void>;
}

const COLORS = [
  { code: 'R', name: 'Red', colorClass: 'bc-color-red' },
  { code: 'B', name: 'Blue', colorClass: 'bc-color-blue' },
  { code: 'G', name: 'Green', colorClass: 'bc-color-green' },
  { code: 'Y', name: 'Yellow', colorClass: 'bc-color-yellow' },
  { code: 'O', name: 'Orange', colorClass: 'bc-color-orange' },
  { code: 'P', name: 'Purple', colorClass: 'bc-color-purple' },
];

const NUMBERS = ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9'];

export default function TwoPlayerBoard({
  userId,
  mode,
  myCode,
  opponentLastGuess,
  myGuesses,
  currentTurn,
  maxGuesses,
  status,
  winner,
  onSubmitGuess,
}: TwoPlayerBoardProps) {
  const codeLength = mode === 'colors' ? 4 : 5;
  const [currentGuess, setCurrentGuess] = useState<string[]>(new Array(codeLength).fill(''));
  const [selectedPosition, setSelectedPosition] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [revealingCode, setRevealingCode] = useState(false);
  const [submittedThisTurn, setSubmittedThisTurn] = useState(false);

  const options = mode === 'colors' ? COLORS : NUMBERS.map(n => ({ code: n, name: n, colorClass: '' }));

  // Check if I've guessed this turn (from server data or local state)
  const myTurnGuess = myGuesses.find(g => g.turnNumber === currentTurn);
  const hasGuessedThisTurn = !!myTurnGuess || submittedThisTurn;

  const handleOptionClick = (value: string) => {
    if (hasGuessedThisTurn || status !== 'active') return;

    // Check for duplicates
    if (currentGuess.includes(value)) {
      setError('No duplicates allowed');
      setTimeout(() => setError(null), 2000);
      return;
    }

    const newGuess = [...currentGuess];
    newGuess[selectedPosition] = value;
    setCurrentGuess(newGuess);

    // Move to next empty position
    const nextEmpty = newGuess.findIndex((c, i) => i > selectedPosition && c === '');
    if (nextEmpty !== -1) {
      setSelectedPosition(nextEmpty);
    } else if (selectedPosition < codeLength - 1) {
      setSelectedPosition(selectedPosition + 1);
    }
  };

  const handlePositionClick = (index: number) => {
    if (!hasGuessedThisTurn && status === 'active') {
      setSelectedPosition(index);
    }
  };

  const handleClear = () => {
    if (!hasGuessedThisTurn && status === 'active') {
      setCurrentGuess(new Array(codeLength).fill(''));
      setSelectedPosition(0);
      setError(null);
    }
  };

  const handleSubmit = async () => {
    if (currentGuess.some(c => c === '')) {
      setError('Complete your guess first');
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      await onSubmitGuess(currentGuess.join(''));
      setSubmittedThisTurn(true);
      setCurrentGuess(new Array(codeLength).fill(''));
      setSelectedPosition(0);
    } catch (err: any) {
      setError(err.message || 'Failed to submit guess');
    } finally {
      setSubmitting(false);
    }
  };

  // Reset submitted state when turn changes
  useEffect(() => {
    setSubmittedThisTurn(false);
  }, [currentTurn]);

  const getColorClass = (code: string) => {
    const color = COLORS.find(c => c.code === code);
    return color ? color.colorClass : '';
  };

  const renderCodePeg = (value: string, index: number, isSmall = false) => {
    const option = mode === 'colors' ? COLORS.find(c => c.code === value) : null;
    return (
      <div
        key={index}
        className={`${isSmall ? 'bc-history-peg' : 'bc-peg'} ${mode === 'colors' && option ? 'color-mode ' + option.colorClass : ''}`}
      >
        {mode === 'numbers' ? value : ''}
      </div>
    );
  };

  const renderCompactCodePeg = (value: string, index: number) => {
    const option = mode === 'colors' ? COLORS.find(c => c.code === value) : null;
    return (
      <div
        key={index}
        className={`bc-peg-compact ${mode === 'colors' && option ? 'color-mode ' + option.colorClass : ''}`}
      >
        {mode === 'numbers' ? value : ''}
      </div>
    );
  };

  const renderFeedback = (bulls: number, cows: number) => {
    return (
      <div className="bc-feedback">
        <span className="bc-badge bc-badge--bulls">{bulls} ✓</span>
        <span className="bc-badge bc-badge--cows">{cows} ~</span>
      </div>
    );
  };

  return (
    <>
      <div className="ah-container ah-container--narrow">
        {/* Game Info Bar */}
        <div className="ah-flex-center ah-mb">
          <div className="ah-badge">
            {mode === 'colors' ? 'Colors' : 'Numbers'}
          </div>
          <div className="ah-badge">
            {myGuesses.length} / {maxGuesses} Guesses
          </div>
          <div className="ah-badge">
            Turn {currentTurn}
          </div>
        </div>
        {/* My Code (Privacy Toggle) - Compact for mobile */}
        <div className="ah-card ah-mb">
          <div className="bc-secret-code-compact">
            {revealingCode
              ? myCode.split('').map((value, index) => renderCompactCodePeg(value, index))
              : new Array(codeLength).fill('*').map((_, index) => (
                  <div key={index} className="bc-peg-compact bc-peg-hidden">
                    *
                  </div>
                ))}
            <button
              className="bc-eye-button"
              onMouseDown={() => setRevealingCode(true)}
              onMouseUp={() => setRevealingCode(false)}
              onMouseLeave={() => setRevealingCode(false)}
              onTouchStart={() => setRevealingCode(true)}
              onTouchEnd={() => setRevealingCode(false)}
              title="Press and hold to reveal"
            >
              👁️
            </button>
          </div>
        </div>

        {/* Opponent's Last Guess - Compact for mobile */}
        <div className="ah-card ah-mb">
          <div className="ah-flex-between">
            <div className="ah-meta">
              Opponent's<br />last guess
            </div>
            <div>
              {opponentLastGuess && renderFeedback(opponentLastGuess.bulls, opponentLastGuess.cows)}
            </div>
          </div>
        </div>

        {/* Current Guess Input */}
        {status === 'active' && (
          <div className="ah-card ah-mb">
            <h3>
              Your Guess
              {hasGuessedThisTurn && <span className="ah-meta"> ⏳ waiting</span>}
            </h3>

            <div className={`bc-guess-display ${hasGuessedThisTurn ? 'bc-guess-display-disabled' : ''}`}>
              {currentGuess.map((value, index) => (
                <div
                  key={index}
                  onClick={() => !hasGuessedThisTurn && handlePositionClick(index)}
                  className={`bc-peg ${selectedPosition === index && !hasGuessedThisTurn ? 'selected' : ''} ${
                    value && mode === 'colors' ? 'color-mode ' + getColorClass(value) : ''
                  } ${hasGuessedThisTurn ? 'bc-peg-disabled' : ''}`}
                >
                  {value || '?'}
                </div>
              ))}
            </div>

            {/* Selection Interface */}
            {mode === 'colors' ? (
              <div className="bc-color-grid ah-mt">
                {options.map((option) => (
                  <button
                    key={option.code}
                    onClick={() => handleOptionClick(option.code)}
                    className={`bc-color-btn ${option.colorClass}`}
                    disabled={hasGuessedThisTurn || currentGuess.includes(option.code)}
                  >
                    {option.name}
                  </button>
                ))}
              </div>
            ) : (
              <div className="bc-number-grid ah-mt">
                {options.map((option) => (
                  <button
                    key={option.code}
                    onClick={() => handleOptionClick(option.code)}
                    className="ah-btn-outline bc-number-btn"
                    disabled={hasGuessedThisTurn || currentGuess.includes(option.code)}
                  >
                    {option.code}
                  </button>
                ))}
              </div>
            )}

            {error && (
              <div className="ah-banner ah-banner--error ah-mt">
                {error}
              </div>
            )}

            <div className="ah-flex-between ah-mt">
              <button className="ah-btn-outline" onClick={handleClear} disabled={hasGuessedThisTurn}>
                Clear
              </button>
              <button
                className="ah-btn-primary"
                onClick={handleSubmit}
                disabled={hasGuessedThisTurn || submitting || currentGuess.some(c => c === '')}
              >
                {submitting ? 'Submitting...' : 'Submit Guess'}
              </button>
            </div>
          </div>
        )}

        {/* Game over */}
        {status !== 'active' && status !== 'code_setting' && (
          <div className={`ah-banner ${status === 'won' && winner === userId ? 'ah-banner--success' : 'ah-banner--error'} ah-mb`}>
            {status === 'won' && winner === userId && '🎉 You won!'}
            {status === 'won' && winner !== userId && '😞 Opponent won'}
            {status === 'draw' && '🤝 Draw!'}
          </div>
        )}

        {/* Guess History - Below input */}
        <div className="ah-card">
          <h3>My Guess History</h3>

          {myGuesses.length === 0 ? (
            <p className="ah-meta">No guesses yet</p>
          ) : (
            <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
              {[...myGuesses].reverse().map((guess) => (
                <div key={guess.id} className="ah-list-item ah-mb-sm">
                  <div className="ah-flex-center">
                    <div className="ah-badge">#{guess.turnNumber}</div>
                    <div className="bc-history-pegs">
                      {guess.guessCode.split('').map((value, idx) => renderCodePeg(value, idx, true))}
                    </div>
                    <div className="bc-feedback">
                      <div className="bc-bulls-badge">✓ {guess.bulls}</div>
                      <div className="bc-cows-badge">~ {guess.cows}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </>
  );
}
