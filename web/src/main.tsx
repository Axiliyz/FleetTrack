import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '@fontsource/ibm-plex-sans-condensed/400.css'
import '@fontsource/ibm-plex-sans-condensed/500.css'
import '@fontsource/ibm-plex-sans-condensed/600.css'
import './styles.css'
import { App } from './App'

const root = document.getElementById('root')
if (!root) throw new Error('нет элемента #root в index.html')

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
