import { useMemo } from 'react';
import { useActivityHubContext } from 'activity-hub-sdk';
import GameBoard from './components/GameBoard';

// Parse query params from URL
function useQueryParams() {
  return useMemo(() => {
    const params = new URLSearchParams(window.location.search);
    return {
      gameId: params.get('gameId'),
    };
  }, []);
}

function App() {
  const { gameId } = useQueryParams();
  const { user, roles } = useActivityHubContext();

  // Check authentication
  if (!user || user.isGuest) {
    return (
      <div className="ah-container ah-container--narrow ah-mt-lg ah-text-center">
        <h2>Authentication Required</h2>
        <p className="ah-meta">Please access this game through Activity Hub</p>
      </div>
    );
  }

  // Check player role (should be auto-assigned as default role)
  if (!roles.hasApp('player')) {
    return (
      <div className="ah-container ah-container--narrow ah-mt-lg ah-text-center">
        <h2>Access Denied</h2>
        <p className="ah-meta">
          You don't have permission to play Bulls and Cows.
        </p>
        <p className="ah-meta">Contact an administrator to request access.</p>
      </div>
    );
  }

  // Must have gameId to play
  if (!gameId) {
    return (
      <div className="ah-container ah-container--narrow ah-mt-lg ah-text-center">
        <h2>Game ID Required</h2>
        <p className="ah-meta">No game found. Please start a new game from the lobby.</p>
      </div>
    );
  }

  return <GameBoard gameId={gameId} user={user} />;
}

export default App;
