import { createApp } from 'vue'
import App from './App.vue'
import { applyTheme, readTheme } from './theme'
import { applyUIScale, readUIScale } from './accessibility'
import { applyInterfaceLocale, readInterfaceLocale } from './locale'
import './styles.css'

applyTheme(readTheme(), false)
applyUIScale(readUIScale(), false)
applyInterfaceLocale(readInterfaceLocale(), false)

createApp(App).mount('#app')
