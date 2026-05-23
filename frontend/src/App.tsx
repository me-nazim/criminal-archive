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

import Cases from './routes/Cases'
import CaseDetail from './routes/CaseDetail'
import Persons from './routes/Persons'
import PersonProfile from './routes/PersonProfile'
import Search from './routes/Search'

import MyCases from './routes/MyCases'
import MyCaseNew from './routes/MyCaseNew'
import MyCaseEdit from './routes/MyCaseEdit'

import AdminLayout from './routes/admin/AdminLayout'
import AdminDashboard from './routes/admin/Dashboard'
import AdminApprovals from './routes/admin/Approvals'
import AdminUsers from './routes/admin/Users'
import AdminCases from './routes/admin/Cases'
import AdminCaseEdit from './routes/admin/CaseEdit'
import AdminPersons from './routes/admin/Persons'
import AdminVerification from './routes/admin/Verification'
import AdminAuditLog from './routes/admin/AuditLog'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Home />} />

        {/* Public read */}
        <Route path="cases" element={<Cases />} />
        <Route path="cases/:key" element={<CaseDetail />} />
        <Route path="persons" element={<Persons />} />
        <Route path="persons/:slug" element={<PersonProfile />} />
        <Route path="search" element={<Search />} />

        {/* Public auth */}
        <Route path="login" element={<Login />} />
        <Route path="register" element={<Register />} />
        <Route path="register/pending" element={<RegisterPending />} />

        {/* Authenticated user */}
        <Route
          path="me"
          element={
            <RequireAuth>
              <Me />
            </RequireAuth>
          }
        />
        <Route
          path="me/cases"
          element={
            <RequireAuth>
              <MyCases />
            </RequireAuth>
          }
        />
        <Route
          path="me/cases/new"
          element={
            <RequireAuth>
              <MyCaseNew />
            </RequireAuth>
          }
        />
        <Route
          path="me/cases/:id/edit"
          element={
            <RequireAuth>
              <MyCaseEdit />
            </RequireAuth>
          }
        />

        {/* Admin */}
        <Route
          path="admin"
          element={
            <RequireAuth minRole="admin">
              <AdminLayout />
            </RequireAuth>
          }
        >
          <Route index element={<AdminDashboard />} />
          <Route path="approvals" element={<AdminApprovals />} />
          <Route path="users" element={<AdminUsers />} />
          <Route path="cases" element={<AdminCases />} />
          <Route path="cases/:id" element={<AdminCaseEdit />} />
          <Route path="persons" element={<AdminPersons />} />
          <Route path="verification" element={<AdminVerification />} />
          <Route path="audit-log" element={<AdminAuditLog />} />
        </Route>

        {/* Misc */}
        <Route path="forbidden" element={<Forbidden />} />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}
