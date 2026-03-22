import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';

// Import game-specific board CSS only
import './styles/bulls-and-cows-board.css';

// NOTE: Shared Activity Hub CSS is auto-loaded by activity-hub-sdk
// No manual CSS injection needed

const root = ReactDOM.createRoot(document.getElementById('root')!);
root.render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>
);
