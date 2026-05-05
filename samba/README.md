# For Little Samba AD DC

Samba Active Directory Domain Controller for Windows clients that need domain join and Group Policy. This is intended to support For Little's local management model:

- Windows clients join the domain through VPN.
- Chrome policy is delivered by GPO instead of `--load-extension`.
- The Windows agent remains a watchdog for profile/process enforcement.

## Important Constraints

- Do not expose AD DC ports directly to the public internet.
- Put Windows clients and the DC on a private network, preferably WireGuard/Tailscale.
- Use a dedicated Ubuntu VPS or VM for the DC role. Avoid running file/print services on the AD DC.
- Windows clients must be Pro, Education, or Enterprise to join a domain.
- `network_mode: host` is intentional because AD DC needs DNS, Kerberos, LDAP, SMB, and RPC behavior that is fragile through normal Docker port publishing.
- `CAP_SYS_ADMIN` is intentionally added because Samba AD DC needs to set NT ACL metadata on `SYSVOL/NETLOGON` during provision. Without it, provisioning commonly fails at `set_nt_acl`.

## Domain Defaults

Recommended:

```text
Realm:   AD.HATBUINHO.ME
NetBIOS: FORLITTLE
Host:    dc1.ad.hatbuinho.me
```

Avoid `.local` because it conflicts with mDNS/Bonjour.

## Setup

Create env:

```bash
cp .env.example .env
```

Edit `.env`:

```env
SAMBA_REALM=AD.HATBUINHO.ME
SAMBA_DOMAIN=FORLITTLE
SAMBA_HOSTNAME=dc1
SAMBA_HOST_IP=
SAMBA_INTERFACES=
SAMBA_BIND_INTERFACES_ONLY=no
SAMBA_ADMIN_PASSWORD=replace-with-strong-password
SAMBA_DNS_FORWARDER=1.1.1.1
SAMBA_LOG_LEVEL=1
```

Set `SAMBA_HOST_IP` to the VPN/private IP that Windows clients will use for the DC if the server has multiple interfaces.
Leave `SAMBA_INTERFACES` empty unless you need to override it.
Keep `SAMBA_BIND_INTERFACES_ONLY=no` for Tailscale. Samba's own documentation warns that `bind interfaces only = yes` does not cope well with PPP or other non-broadcast interfaces, and `tailscale0` is point-to-point. If you force bind-only mode, Samba may listen only on loopback or IPv6.

When `SAMBA_BIND_INTERFACES_ONLY=yes`, the entrypoint defaults `SAMBA_INTERFACES` to `127.0.0.1` plus `SAMBA_HOST_IP`.

Start:

```bash
docker compose up -d --build
```

If a previous provision failed halfway, remove the volumes before retrying:

```bash
docker compose down -v
docker compose up -d --build
```

The container keeps its own provision marker. If Samba files exist without that marker, the entrypoint treats them as incomplete state and reprovisions from scratch.

If `systemd-resolved` is holding port `53`, disable the local DNS stub listener on the host before starting Samba:

```bash
sudo sed -i 's/^#\\?DNSStubListener=.*/DNSStubListener=no/' /etc/systemd/resolved.conf
sudo systemctl restart systemd-resolved
sudo ss -tulpn | grep ':53'
```

After that, Samba should be the only DNS server you intend to expose for AD traffic. If you run with `SAMBA_BIND_INTERFACES_ONLY=no`, restrict exposure with host firewall rules and Tailscale ACLs instead of relying on Samba socket binding alone.

Check:

```bash
./scripts/check-dc.sh
```

Create domain user:

```bash
./scripts/create-user.sh little01
```

## Windows Client Join Flow

1. Connect the Windows client to the VPN.
2. Set the Windows DNS server to the DC VPN IP.
3. Verify DNS:

```powershell
nslookup -type=SRV _ldap._tcp.ad.hatbuinho.me
nslookup dc1.ad.hatbuinho.me
```

4. Join domain:

```powershell
Add-Computer -DomainName "AD.HATBUINHO.ME" -Credential "FORLITTLE\Administrator" -Restart
```

5. After restart, login as a domain user:

```text
FORLITTLE\little01
```

## Chrome GPO Plan

Use a Windows admin machine joined to the domain:

1. Install RSAT Group Policy Management tools.
2. Download Chrome Enterprise ADMX templates.
3. Copy ADMX/ADML to the domain Central Store:

```text
\\ad.hatbuinho.me\SYSVOL\ad.hatbuinho.me\Policies\PolicyDefinitions
```

4. Create GPO for For Little machines.
5. Configure Chrome extension policy:

```text
Computer Configuration
Administrative Templates
Google
Google Chrome
Extensions
```

For self-host CRX, prefer `ExtensionSettings`:

```json
{
  "EXTENSION_ID": {
    "installation_mode": "force_installed",
    "update_url": "https://little-be.hatbuinho.me/extensions/forlittle/update.xml",
    "override_update_url": true
  }
}
```

Verify on client:

```text
chrome://policy
chrome://extensions
```

## ForLittle Windows Agent Role

After domain/GPO exists, the agent should stop using `--load-extension`. It should:

- Launch Chrome with the managed profile.
- Relaunch Chrome if closed.
- Kill Chrome with unmanaged profile.
- Optionally check policy presence.
- Optionally block other browsers.

## Backup

Run:

```bash
./scripts/backup.sh
```

At minimum, protect Samba state under:

```text
/var/lib/samba
/etc/samba
```

Losing this data means losing the domain identity.
