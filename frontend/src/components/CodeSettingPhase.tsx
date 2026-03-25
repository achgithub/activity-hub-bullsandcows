import { useState } from 'react';

interface User {
  email: string;
  name: string;
}

interface CodeSettingPhaseProps {
  gameId: string;
  user: User;
  mode: string; // 'colors' or 'numbers'
  myCodeSet: boolean;
  opponentCodeSet: boolean;
  onCodeSet: () => void;
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

export default function CodeSettingPhase({
  gameId,
  user,
  mode,
  myCodeSet,
  opponentCodeSet,
  onCodeSet,
}: CodeSettingPhaseProps) {
  const codeLength = mode === 'colors' ? 4 : 5;
  const [code, setCode] = useState<string[]>(new Array(codeLength).fill(''));
  const [selectedPosition, setSelectedPosition] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const options = mode === 'colors' ? COLORS : NUMBERS.map(n => ({ code: n, name: n, colorClass: '' }));

  const getColorClass = (code: string) => {
    const color = COLORS.find(c => c.code === code);
    return color ? color.colorClass : '';
  };

  const handleOptionClick = (value: string) => {
    if (myCodeSet) return; // Can't change after submitting

    // Check for duplicates
    if (code.includes(value)) {
      setError('No duplicates allowed');
      setTimeout(() => setError(null), 2000);
      return;
    }

    const newCode = [...code];
    newCode[selectedPosition] = value;
    setCode(newCode);

    // Move to next empty position
    const nextEmpty = newCode.findIndex((c, i) => i > selectedPosition && c === '');
    if (nextEmpty !== -1) {
      setSelectedPosition(nextEmpty);
    } else if (selectedPosition < codeLength - 1) {
      setSelectedPosition(selectedPosition + 1);
    }
  };

  const handlePositionClick = (index: number) => {
    if (!myCodeSet) {
      setSelectedPosition(index);
    }
  };

  const handleClear = () => {
    if (!myCodeSet) {
      setCode(new Array(codeLength).fill(''));
      setSelectedPosition(0);
      setError(null);
    }
  };

  const handleSubmit = async () => {
    if (code.some(c => c === '')) {
      setError('Complete the code first');
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const token = localStorage.getItem('token');
      if (!token) {
        throw new Error('No authentication token');
      }

      const response = await fetch(`/api/apps/bulls-and-cows/proxy/api/game/${gameId}/set-code`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-User-ID': user.email,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ code: code.join('') }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'Failed to set code');
      }

      onCodeSet();
    } catch (err: any) {
      console.error('Error setting code:', err);
      setError(err.message || 'Failed to set code');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="ah-container ah-container--narrow ah-mt">
      <div className="ah-card">
        <h2>Set Your Secret Code</h2>

        <p className="ah-meta ah-mb-lg">
          Choose a secret code for your opponent to crack. No duplicates allowed.
        </p>

        {/* Code display */}
        <div className="bc-guess-display">
          {code.map((value, index) => (
            <div
              key={index}
              onClick={() => !myCodeSet && handlePositionClick(index)}
              className={`bc-peg ${selectedPosition === index && !myCodeSet ? 'selected' : ''} ${
                value && mode === 'colors' ? 'color-mode ' + getColorClass(value) : ''
              } ${myCodeSet ? 'bc-peg-set' : ''}`}
            >
              {value || '?'}
            </div>
          ))}
        </div>

        {/* Option selector */}
        {!myCodeSet && (
          <>
            {mode === 'colors' ? (
              <div className="bc-color-grid ah-mt">
                {options.map((option) => (
                  <button
                    key={option.code}
                    onClick={() => handleOptionClick(option.code)}
                    className={`bc-color-btn ${option.colorClass}`}
                    disabled={code.includes(option.code)}
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
                    disabled={code.includes(option.code)}
                  >
                    {option.code}
                  </button>
                ))}
              </div>
            )}
          </>
        )}

        {/* Error message */}
        {error && (
          <div className="ah-banner ah-banner--error ah-mb">
            {error}
          </div>
        )}

        {/* Status */}
        {myCodeSet ? (
          <div className="ah-banner ah-banner--success ah-mt">
            ✓ Your code is set. {opponentCodeSet ? 'Game starting!' : 'Waiting for opponent...'}
          </div>
        ) : (
          <div className="ah-flex-between ah-mt">
            <button className="ah-btn-outline" onClick={handleClear}>
              Clear
            </button>
            <button
              className="ah-btn-primary"
              onClick={handleSubmit}
              disabled={submitting || code.some(c => c === '')}
            >
              {submitting ? 'Submitting...' : 'Set Code'}
            </button>
            <button className="ah-btn-outline" onClick={handleClear}>
              Clear
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
