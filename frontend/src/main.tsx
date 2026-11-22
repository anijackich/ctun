import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './App'
import { installPressFx } from './lib/fx'
import { installInputModality } from './lib/modality'
import './styles/tokens.css'
import './styles/base.css'
import './styles/icons.css'

installPressFx()
installInputModality()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
