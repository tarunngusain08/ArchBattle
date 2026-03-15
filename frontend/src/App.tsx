import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { AuthGuard } from './components/auth/AuthGuard'
import { AppLayout } from './components/layout/AppLayout'
import { DailyChallengePage } from './pages/DailyChallenge'
import { BattlePage } from './pages/Battle'
import { HomePage } from './pages/Home'
import { LobbyPage } from './pages/Lobby'
import { LoginPage } from './pages/Login'
import { ProfilePage } from './pages/Profile'
import { QueuePage } from './pages/Queue'
import { DiscussionPage } from './pages/Discussion'
import { ResultsPage } from './pages/Results'
import { RevealPage } from './pages/Reveal'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        {/* All routes inside AppLayout require authentication. */}
        <Route element={<AuthGuard />}>
          <Route element={<AppLayout />}>
            <Route path="/" element={<HomePage />} />
            <Route path="/queue" element={<QueuePage />} />
            <Route path="/lobby" element={<LobbyPage />} />
            <Route path="/battle" element={<BattlePage />} />
            <Route path="/reveal" element={<RevealPage />} />
            <Route path="/results" element={<ResultsPage />} />
            <Route path="/daily" element={<DailyChallengePage />} />
            <Route path="/discussion" element={<DiscussionPage />} />
            <Route path="/profile" element={<ProfilePage />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
