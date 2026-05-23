import { Route, Routes } from 'react-router-dom'
import Layout from './components/layout/Layout'
import Home from './routes/Home'
import NotFound from './routes/NotFound'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Home />} />
        {/* TODO: /cases, /cases/:slug, /persons, /persons/:slug,
                  /submit, /login, /register, /admin/* */}
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}
