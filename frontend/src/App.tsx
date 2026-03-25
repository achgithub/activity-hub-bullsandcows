import { useMemo } from 'react';
import { AppHeader, GameCard, useActivityHubContext } from 'activity-hub-sdk';
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
      <>
        <AppHeader title="Bulls and Cows" icon="🐂" />
        <GameCard size="narrow">
          <p className="ah-meta">
            Authentication required. Please access this game through Activity Hub.
          </p>
        </GameCard>
      </>
    );
  }

  // Check player role (should be auto-assigned as default role)
  if (!roles.hasApp('player')) {
    return (
      <>
        <AppHeader title="Bulls and Cows" icon="🐂" />
        <GameCard size="narrow">
          <p className="ah-meta">
            You don't have permission to play Bulls and Cows.
          </p>
          <p className="ah-meta">
            Contact an administrator to request access.
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

  return <GameBoard gameId={gameId} user={user} />;
}

export default App;
