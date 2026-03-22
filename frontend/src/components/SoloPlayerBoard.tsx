import React, { useState } from 'react';

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

interface SoloPlayerBoardProps {
  gameId: string;
  token: string;
  userId: string;
  mode: string;
  secretCode?: string;
  guesses: Guess[];
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

export default function SoloPlayerBoard({
  gameId,
  token,
  userId,
  mode,
  secretCode,
  guesses,
  maxGuesses,
  status,
  winner,
  onSubmitGuess,
}: SoloPlayerBoardProps) {
  const codeLength = mode === 'colors' ? 4 : 5;
  const [currentGuess, setCurrentGuess] = useState<string[]>(new Array(codeLength).fill(''));
  const [selectedPosition, setSelectedPosition] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const options = mode === 'colors' ? COLORS : NUMBERS.map(n => ({ code: n, name: n, colorClass: '' }));
  const isGameOver = status !== 'active';

  const handleOptionClick = (value: string) => {
    if (isGameOver) return;

    // Check for duplicates
    if (currentGuess.includes(value)) {
      setError('No duplicates allowed');
      setTimeout(() => setError(null), 2000);
      return;
    }

    setError(null);
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
    if (!isGameOver) {
      setSelectedPosition(index);
    }
  };

  const handleClear = () => {
    if (!isGameOver) {
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
      setCurrentGuess(new Array(codeLength).fill(''));
      setSelectedPosition(0);
    } catch (err: any) {
      setError(err.message || 'Failed to submit guess');
    } finally {
      setSubmitting(false);
    }
  };

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
      {/* Game Info Bar */}
      <div className="bc-game-info-bar">
        <div className="bc-game-info-content">
          <div className="ah-badge">
            {mode === 'colors' ? 'Colors' : 'Numbers'}
          </div>
          <div className="ah-badge">
            {guesses.length} / {maxGuesses} Guesses
          </div>
        </div>
      </div>

      <div className="ah-container ah-container--narrow ah-mt">
        {/* Game Over Banner */}
        {isGameOver && (
          <div className={`ah-banner ${status === 'won' ? 'ah-banner--success' : 'ah-banner--error'} ah-mb`}>
            {status === 'won' ? (
              <>🎉 You cracked the code: <strong>{secretCode}</strong></>
            ) : (
              <>😔 The code was: <strong>{secretCode}</strong></>
            )}
          </div>
        )}

        {/* Current Guess Input */}
        {!isGameOver && (
          <div className="ah-card ah-mb">
            <h3 className="bc-section-title">Your Guess</h3>

            <div className="bc-guess-display">
              {currentGuess.map((value, index) => (
                <div
                  key={index}
                  onClick={() => handlePositionClick(index)}
                  className={`bc-peg ${selectedPosition === index ? 'selected' : ''} ${
                    value && mode === 'colors' ? 'color-mode ' + getColorClass(value) : ''
                  }`}
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
                  disabled={currentGuess.includes(option.code)}
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
                  disabled={currentGuess.includes(option.code)}
                >
                  {option.code}
                </button>
              ))}
            </div>
          )}

          {error && (
            <div className="ah-banner ah-banner--error ah-mb">
              {error}
            </div>
          )}

          <div className="bc-actions ah-mt">
            <button
              className="ah-btn-primary"
              onClick={handleSubmit}
              disabled={submitting || currentGuess.some(c => c === '')}
            >
              {submitting ? 'Submitting...' : 'Submit Guess'}
            </button>
            <button className="ah-btn-outline" onClick={handleClear}>
              Clear
            </button>
          </div>
        </div>
      )}

        {/* Guess History */}
        <div className="ah-card">
          <h3 className="bc-section-title">Guess History</h3>

          {guesses.length === 0 ? (
            <p className="ah-meta">No guesses yet</p>
          ) : (
            <div className="bc-history-scroll">
              {[...guesses].reverse().map(guess => (
                <div key={guess.id} className="ah-list-item ah-mb-sm">
                  <div className="bc-history-item-content">
                    <div className="bc-guess-number">
                      #{guess.turnNumber}
                    </div>
                    <div className="bc-history-pegs">
                      {guess.guessCode.split('').map((value, idx) => renderCodePeg(value, idx, true))}
                    </div>
                    <div className="bc-feedback">
                      <div className="bc-bulls-badge">
                        ✓ {guess.bulls}
                      </div>
                      <div className="bc-cows-badge">
                        ~ {guess.cows}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Legend */}
        <div className="ah-card bc-legend ah-mt">
          <p className="ah-mb-none">
            <strong>Legend:</strong> ✓ Bulls (correct position) | ~ Cows (wrong position)
          </p>
        </div>
      </div>
    </>
  );
}
