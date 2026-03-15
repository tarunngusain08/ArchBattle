import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { PlayerGuard } from './components/auth/PlayerGuard'
import { AppLayout } from './components/layout/AppLayout'
import { DailyChallengePage } from './pages/DailyChallenge'
import { BattlePage } from './pages/Battle'
import { EnterPage } from './pages/Enter'
import { LobbyPage } from './pages/Lobby'
import { RoomsPage } from './pages/Rooms'
import { ResultsPage } from './pages/Results'
import { RevealPage } from './pages/Reveal'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<EnterPage />} />
        <Route element={<PlayerGuard />}>
          <Route element={<AppLayout />}>
            <Route path="/rooms" element={<RoomsPage />} />
            <Route path="/lobby" element={<LobbyPage />} />
            <Route path="/battle" element={<BattlePage />} />
            <Route path="/reveal" element={<RevealPage />} />
            <Route path="/results" element={<ResultsPage />} />
            <Route path="/daily" element={<DailyChallengePage />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
