# Local Network / iPhone Testing

Use this mode when you want to open the local Go server from a phone on the same Wi-Fi network.

1. Make sure the computer and phone are connected to the same Wi-Fi.
2. Find the computer IPv4 address on the Wi-Fi adapter:

   ```powershell
   ipconfig
   ```

3. Run the server on all local interfaces:

   ```powershell
   $env:HOST="0.0.0.0"
   $env:PORT="8080"
   $env:COOKIE_SECURE="false"
   go run ./cmd/server
   ```

4. On the phone, open:

   ```text
   http://LOCAL_PC_IP:8080
   ```

Use `http`, not `https`, for local development. If Windows Firewall asks for access, allow the app on the private network. In production, HTTPS should be provided by the deploy platform or domain, and `COOKIE_SECURE=true` should only be enabled behind HTTPS.
