import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'

import App from './App'
import { queryClient } from './lib/query-client'
import { bootstrapAuth } from './lib/auth-bootstrap'
import './lib/i18n'
import './styles/index.css'

// Kick off auth bootstrap synchronously; the auth store has a
// `bootstrapped` flag that route guards observe to avoid flashing
// the login page on a hard reload.
void bootstrapAuth()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)
