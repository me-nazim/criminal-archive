import { Route, Routes } from 'react-router-dom'

import Layout from './components/layout/Layout'
import { RequireAuth } from './components/RequireAuth'

import Home from './routes/Home'
import NotFound from './routes/NotFound'
import Forbidden from './routes/Forbidden'
import Login from './routes/Login'
import Register from './routes/Register'
import RegisterPending from './routes/RegisterPending'
import Me from './routes/Me'

import AdminLayout from './routes/admin/AdminLayout'
import Dashboard from './routes/admin/Dashboard'
import Approvals from './routes/admin/Approvals'
import Users from './routes/admin/Users'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Home />} />

        {/* Public auth routes */}
        <Route path="login" element={<Login />} />
        <Route path="register" element={<Register />} />
        <Route path="register/pending" element={<RegisterPending />} />

        {/* Authenticated user routes */}
        <Route
          path="me"
          element={
            <RequireAuth>
              <Me />
            </RequireAuth>
          }
        />

        {/* Admin routes (admin or super-admin) */}
        <Route
          path="admin"
          element={
            <RequireAuth minRole="admin">
              <AdminLayout />
            </RequireAuth>
          }
        >
          <Route index element={<Dashboard />} />
          <Route path="approvals" element={<Approvals />} />
          <Route path="users" element={<Users />} />
        </Route>

        {/* Misc */}
        <Route path="forbidden" element={<Forbidden />} />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}
