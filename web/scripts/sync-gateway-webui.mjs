import { cpSync, mkdirSync, rmSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const currentDir = dirname(fileURLToPath(import.meta.url))
const webRoot = resolve(currentDir, '..')
const distDir = resolve(webRoot, 'dist')
const gatewayDir = resolve(webRoot, '..', 'internal', 'gateway', 'webui')

rmSync(gatewayDir, { recursive: true, force: true })
mkdirSync(gatewayDir, { recursive: true })
cpSync(distDir, gatewayDir, { recursive: true })
console.log(`Synced ${distDir} -> ${gatewayDir}`)
