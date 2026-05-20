# Vendored Browser Libraries

## nats.ws

- Package: `nats.ws`
- Version: `1.30.3`
- Source: npm package `nats.ws@1.30.3`
- Files:
  - `nats.ws.js` from `package/esm/nats.js`
  - `nats.ws.LICENSE.txt` from `package/LICENSE`

The browser UI uses this vendored ESM bundle so Palette can use NATS over
WebSocket without adding an npm install/build step to the web UI.

To update:

```powershell
$tmp = Join-Path $env:TEMP ('natsws_' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp | Out-Null
npm pack nats.ws@VERSION --pack-destination $tmp
tar -xf (Get-ChildItem $tmp -Filter *.tgz | Select-Object -First 1).FullName -C $tmp package/esm/nats.js package/LICENSE
Copy-Item "$tmp\package\esm\nats.js" kit\webui\vendor\nats.ws.js -Force
Copy-Item "$tmp\package\LICENSE" kit\webui\vendor\nats.ws.LICENSE.txt -Force
```
