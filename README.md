# Mrs Santa

Santa sync server inspired by projects like Rudolph and Moroz

### Goals

* "Integrate" with Munki, Jamf, Kandji etc, so managed software is only allowed to run if installed as a managed app. This is to keep apps patched and be in control of licensing
* Frequent syncs (eg. every 60 secs) to allow close to live blocking/unblocking

### What is it

It's a set of small containers that run on Kubernetes behind a single ingress:

* **Sync services**, written in Go: `preflight`, `eventupload`, `ruledownload` and `postflight` implement Santa's sync protocol, and `xsrf` hands out the optional XSRF tokens
* **`api`**, a Go REST API for automation. The scripts in `tools/` use it, e.g. to create rules for managed apps
* **`metrics`**, a Go endpoint that takes Santa's metric uploads and writes them to the log
* **`worker`**, a Go service for background jobs. Each job is turned on from the admin settings. So far there's one: checking the binaries in your events against MalwareBazaar's list of macOS malware
* **`admin`**, a Python (Flask) web UI for managing devices, rules, settings and events
* **`web`**, a Python (Flask) site that Santa's block dialog links to, where users can ask for a blocked app to be allowed

Data is persisted in MongoDB (MongoDB Atlas in the Terraform examples), and each web container serves its own icons and images. Admins and users sign in with Microsoft Entra ID; other identity providers aren't supported yet (see [Sign-in](#sign-in)).

The busiest sync services scale automatically with a horizontal pod autoscaler. Our Mac fleet is fairly small, so I can't speak to how far it goes, but that part is completely hands off.

### Deploying to Kubernetes

`deploy/` has example Terraform for Google Cloud (GKE), Azure (AKS) and AWS (EKS). **They are examples, not a turnkey installer.** They are written for one particular setup and will probably need adapting to your environment, e.g. a different ingress controller, DNS setup, secret store or way of authenticating to the cluster. Read them before applying.

All three assume:

- **A Kubernetes cluster that already exists.** They don't create clusters or networks; they look up the cluster by name and deploy into it.
- **Cluster add-ons already installed:** an ingress controller ([Traefik](https://traefik.io/) by default), [cert-manager](https://cert-manager.io/) with a `ClusterIssuer` for the TLS certificate, and [External Secrets Operator](https://external-secrets.io/) with a `ClusterSecretStore` that reads the cloud's secret store. The app's secrets are written to the cloud's secret store and copied into the cluster by External Secrets Operator.
- **MongoDB Atlas.** They create the database user and the Atlas Search indexes on an existing Atlas cluster, and expect the connection string to be supplied. Using another MongoDB (self-hosted, Azure Cosmos DB, Amazon DocumentDB) means removing the `atlas` module and checking that the database supports what the app uses; the admin UI's search relies on Atlas Search.

Layout:

| Directory | What it does |
|---|---|
| `deploy/modules/app` | The app itself: namespace, deployments, services, autoscalers, the `ExternalSecret`, the certificate and the ingress. Shared by all three clouds |
| `deploy/modules/atlas` | MongoDB Atlas database user and search indexes |
| `deploy/gke` | Google Cloud: Artifact Registry, Secret Manager, Cloud DNS, workload identity. Runs production and a test copy at `test.<service>.<domain>` |
| `deploy/aks` | Azure: resource group, Container Registry (with pull access for the cluster), Key Vault, Azure DNS zone |
| `deploy/eks` | AWS: ECR repositories, Secrets Manager secrets (with a read policy for External Secrets Operator), Route 53 zone |

Each cloud directory creates a DNS zone for `<service>.<domain>` pointing at the ingress, which you delegate from the parent domain (the AKS and EKS examples output the name servers). On AKS and EKS, set `enable_test_environment = true` for a test copy like GKE's. The test copy shares production's database, so it doesn't run the worker.

To deploy:

1. Put the variables in `terraform.auto.tfvars` in the cloud's directory (git ignores it). The variables are described in each directory's `*variables.tf` and `main.tf`.
2. AKS and EKS keep their state in a storage account or S3 bucket you create first. Put its settings in a backend file, e.g. `deploy/eks/backend.hcl` with `bucket`, `key` and `region`, or `deploy/aks/backend.hcl` with `resource_group_name`, `storage_account_name`, `container_name` and `key`. GKE's state bucket is set in `deploy/gke/backend.tf`.
3. From the repository root, run `make init`, `make plan` and `make deploy`. They use `deploy/gke` unless you pass another directory, e.g. `make init TF_DIR=deploy/eks BACKEND_CONFIG=backend.hcl`, then `make plan TF_DIR=deploy/eks`.
4. Build and push the images with `make` in each folder under `containers/`. They push to the GKE registry by default; for the others pass the `image_registry` output, e.g. `make REGISTRY=123456789012.dkr.ecr.eu-west-1.amazonaws.com/mrssanta`. Log in to the registry first (`az acr login`, `aws ecr get-login-password`).
5. Point Santa's `SyncBaseURL` at the `sync_base_url` output.

On AKS, the Kubernetes provider uses the cluster's local accounts; clusters that only allow Entra ID sign-in need an `exec` block running `kubelogin` instead. On EKS, the nodes need read access to ECR, which managed node groups have by default.

The first `make plan` on GKE after updating shows the Kubernetes resources and the Atlas resources being moved into the modules (`moved` blocks in `deploy/gke/app.tf` and `mongodb.tf`), not destroyed and recreated. Check that before applying. The admin and approval site get a new session key in the test copy, which signs test users out once.

Deployment shouldn't take more than 15-20 mins.

### Running locally

The Docker Compose setup in `deploy/compose` runs every service, a MongoDB and a Caddy proxy that routes paths the same way as the Kubernetes ingress. It needs Docker with Compose v2.

All settings and secrets go in `deploy/compose/docker-compose.override.yml`. Compose loads that file automatically on top of `docker-compose.yml`, and git ignores it. Create it from the example and fill in the values:

```sh
cd deploy/compose
cp docker-compose.override.yml.example docker-compose.override.yml
# edit docker-compose.override.yml
```

Then, still in `deploy/compose`:

```sh
docker compose up -d --build   # build the images and start everything
docker compose ps              # check that all containers are running
docker compose logs -f web     # follow one service's logs
docker compose down            # stop everything, keeping the database
docker compose down -v         # stop everything and delete the database
```

To run the commands from the repository root instead, add `--project-directory deploy/compose`, e.g. `docker compose --project-directory deploy/compose up -d --build`.

Once it's running:

* Point Santa's `SyncBaseURL` at `http://localhost:8080/santa/`
* The admin UI is at `http://localhost:8080/admin`
* MongoDB is reachable from the host on `127.0.0.1:27017`. Its data is kept in the `mongo-data` volume
* [Mongo Express](https://github.com/mongo-express/mongo-express), a web UI for browsing and editing the database, is at `http://localhost:8081`. Log in with `ME_CONFIG_BASICAUTH_USERNAME` and `ME_CONFIG_BASICAUTH_PASSWORD` from the override file

Rebuild after changing code with `docker compose up -d --build`, or `docker compose up -d --build <service>` for one service.

The override file sets the variables described below, plus:

| Variable | Services | Description |
|---|---|---|
| `API_KEY` | api, sync services, xsrf | Key Santa and API clients send in the `X-API-Key` header |
| `FLASK_SECRET_KEY` | admin, web | Key used to sign session cookies. Set a different value for each |
| `MSAL_TENANT_ID` | admin, web | Entra ID tenant used for login |
| `MSAL_CLIENT_ID`, `MSAL_CLIENT_SECRET` | admin, web | Each service's own Entra ID app registration (see [Sign-in](#sign-in)). Add `http://localhost:8080/admin/token` as a redirect URI on the admin app and `http://localhost:8080/santa/token` on the approval site's app |
| `MSAL_REDIRECT_URI` | admin, web | Login callback URL. Preset for `localhost:8080` |
| `VT_API_KEY` | admin, web | Optional VirusTotal API key |
| `ORGANIZATION`, `SUPPORT_EMAIL`, `SUPPORT_PHONE` | web | Shown on the block page |
| `VOTE_THRESHOLD` | web | Distinct users who must request an app before its rule becomes global |
| `BATCH_SIZE` | ruledownload | Rules sent per ruledownload response |
| `LOGSTASH_TCP_ADDR` | eventupload | Optional `host:port` events are also forwarded to |
| `ME_CONFIG_BASICAUTH_USERNAME`, `ME_CONFIG_BASICAUTH_PASSWORD` | mongo-express | Login for the database UI |
| `ME_CONFIG_SITE_COOKIESECRET`, `ME_CONFIG_SITE_SESSIONSECRET` | mongo-express | Keys used to sign the database UI's cookies and sessions. Set them to random strings |

### Configuration

The preflight service reads these environment variables. Each one is set from the Terraform variable of the same name in lower case (e.g. `clean_sync_max_age_days`).

| Variable | Default | Description |
|---|---|---|
| `SYNC_INTERVAL_LOCKDOWN` | `60` | Seconds between full syncs for devices in Lockdown mode |
| `SYNC_INTERVAL_MONITOR` | `14400` | Seconds between full syncs for devices in Monitor mode |
| `CLEAN_SYNC_MAX_AGE_DAYS` | `30` | Days a device may go without a completed clean sync before a full (`CLEAN_ALL`) sync is forced, which replaces all of its rules. Must be a whole number of days greater than 0; any other value falls back to the default. The admin service reads it too: the dashboard counts devices that haven't synced within this many days |

The admin and web services also read `SESSION_FILE_DIR`, the directory for login sessions. It defaults to `flask_session` next to the app and normally doesn't need changing.

#### Sign-in

**Only Microsoft Entra ID is supported for now.** Both apps sign in with Microsoft's MSAL library and read the user's details from Microsoft Graph, so other identity providers (Okta, Google, Keycloak and so on) won't work, even though they support OIDC. Generic OIDC support may come later.

The admin UI and the user-facing approval site use two separate Entra ID app registrations: Terraform variables `admin_msal_client_id` / `admin_msal_client_secret` for the admin, and `msal_client_id` / `msal_client_secret` for the approval site. Neither app checks group membership itself: anyone the app registration lets sign in to the admin is a full admin, and anyone who can sign in to the approval site can request access for their own Mac. Restrict them in Entra by enabling **Assignment required** on each app's enterprise application and assigning only the intended users or groups. Admin sign-in should be limited to the people who manage Santa.

#### XSRF tokens

The `xsrf` service answers Santa's `POST /santa/xsrf/{machine_id}` with a signed token in the `X-XSRF-TOKEN` header. Tokens are bound to the machine ID and valid for an hour; Santa fetches one per sync. The sync services (preflight, eventupload, ruledownload, postflight) only check tokens when enforcement is on. A missing or invalid token then gets a 403, which makes Santa fetch a token and retry.

| Variable | Default | Description |
|---|---|---|
| `XSRF_REQUIRED` | `false` | Set to `true` to require a valid token on every sync request. Set from the Terraform variable `xsrf_required` |
| `XSRF_SECRET` | generated | Key used to sign and verify tokens. Terraform generates it and stores it in the cloud's secret store as `<service>-xsrf-secret`. Required by the xsrf service, and by the sync services when `XSRF_REQUIRED` is `true` |

Enforcement only adds protection if Santa authenticates with credentials a browser could also send, such as client certificates. With the `X-API-Key` header alone it isn't needed.

#### Device settings

Everything else Santa receives at preflight is set in the admin UI under **Settings**, and can be overridden per device from the device's edit dialog on the **Devices** page. An empty field means "not set": a device override falls back to the global value, and the global value falls back to the default below or to Santa's own default. Global settings are stored in the `settings` collection (document `_id: "global"`), device overrides in each device's `settings` field.

| Setting | Default | Description |
|---|---|---|
| Default client mode | `LOCKDOWN` | Mode given to newly enrolled devices: `MONITOR`, `LOCKDOWN` or `STANDALONE`. Global only |
| Event upload batch size | `500` | Events Santa uploads per request |
| Bundle hashing | on | `enable_bundles` |
| Compiler (transitive) rules | on | `enable_transitive_rules`. When off, `ALLOWLIST_COMPILER` rules act as `ALLOWLIST` |
| Upload all events / Skip unknown event upload | Santa default | `enable_all_event_upload` / `disable_unknown_event_upload` |
| Allowed / blocked path regex | not set | ICU regexes applied to executions no rule matches. **Build…** next to each field opens a helper (see below) |
| File access override | not set | `NONE`, `AUDIT_ONLY` or `DISABLE` for File Access Authorization policies |
| Removable media / encrypted removable media | not set | Allow, block, or remount with mount flags such as `rdonly,noexec` |
| Event detail URL / button text | not set | The button on Santa's block dialog. The URL supports Santa placeholders such as `%file_sha%` |

A device's client mode, and a one-off clean sync (`CLEAN_RULES`, `CLEAN_ALL` or `CLEAN_STANDALONE`), are also set from the device's edit dialog. The requested clean sync is sent on the device's next sync and cleared once it has completed.

The server no longer creates allow rules automatically from file paths. It used to add a global `ALLOWLIST` rule for any blocked binary whose name or path contained an entry from a built-in list. Because users choose where they put files, that let anyone get any binary allowed on every Mac. Use **Allowed path regex** instead: Santa applies it on the device to each execution, and it never turns into a rule for other Macs. A regex covering the old list (minus the overly broad `flux`) is:

```
(/node_modules/\.pnpm/|/\.pyenv/versions/|^/opt/homebrew/Cellar/python|^/usr/local/Cellar/python|^/usr/local/microsoft/powershell/7/|/terraform-provider-[^/]+$)
```

Keep it as narrow as possible: any user who can write to a matching path can run anything from there. Allowed executions show up as `ALLOW_SCOPE` and are only uploaded when **Upload all events** is on.

Instead of writing the regex by hand, click **Build…** next to either field. The helper builds the regex from a list of conditions: **Inside folder**, **Path starts with**, **Exact path**, **File name is**, **Path contains**, or a **Custom regex**. A path starting with `~` means that folder in any user's home, e.g. `~/bin` becomes `^/Users/[^/]+/bin/`. It reads the field's current regex back into conditions, keeping any part it can't express as a custom regex, so the regex above opens as six editable conditions. You can test a path against the result before using it. For the allowed regex, it warns about conditions that match folders users can write to, or that match in any folder. The test uses the browser's regex engine, which agrees with Santa's ICU engine for everything the helper generates; custom parts using ICU-only syntax may test differently.

#### Self-service allow requests

When Santa blocks an app, the block dialog links to the web service (`/santa/blockables/<machine>/<sha256>`). A signed-in user can ask for the app to be allowed, and the server decides everything about the resulting rule:

- Users can only request access for their own Mac: the device's `primary_user` must match the part of their Entra UPN before the `@`. Macs without a `primary_user` can't be used to request access.
- Apps VirusTotal reports as malicious are refused, and so are apps that already have a global, managed app or block rule.
- The request creates a machine rule for that Mac only. It uses the app's signing ID when the app is properly signed, otherwise its SHA-256. Other binaries from the same bundle (matched by Santa's bundle hash and the same team ID) are included.
- Each user counts once. A rule becomes global when `VOTE_THRESHOLD` distinct users have requested it. Rules created before this change record no voters, so their earlier requests don't count toward the threshold.

#### MalwareBazaar

The admin can check every binary in **Events** against [MalwareBazaar](https://bazaar.abuse.ch/), abuse.ch's free database of malware samples. It's off until you paste an abuse.ch Auth-Key (free from the [Authentication Portal](https://auth.abuse.ch/)) under **Settings → MalwareBazaar**. Clearing the key with **Turn off** stops all requests to abuse.ch.

While it's on, the `worker` service keeps a copy of MalwareBazaar's macOS (Mach-O) samples in the `malwarebazaar_samples` collection and checks binaries against it locally, so no hashes are sent to abuse.ch:

- It downloads the full dump (`full.csv.zip`, all platforms) once and keeps only the Mach-O samples.
- After that it tops the list up from the recent export (`recent.csv`, the last 48 hours), at most once an hour and only if events have arrived since the last update. New samples are matched against all stored events, so binaries seen before MalwareBazaar listed them are flagged too.
- If no events arrive for more than 46 hours, the recent export no longer covers the gap, so the next update downloads the full dump again and rechecks every event.
- Every minute, binaries from newly arrived events are checked against the saved list.

Matches get a **Malware** badge in **Events** and are listed under **Known malware** on the **Review** page with their malware family. **Block everywhere** adds a global `BLOCKLIST` binary rule (replacing any existing rule for that hash); **Dismiss** leaves the rules alone. Either way the match leaves the review list. The settings card shows the size of the saved list, when it was last updated, and the last error, e.g. a rejected key.

Execution events are stored once per binary, without the Mac or user that ran it, so the review list can't show who was affected. The eventupload service's log lines (and Logstash, if `LOGSTASH_TCP_ADDR` is set) include the executing and logged-in users. The key is stored in the `settings` collection (document `_id: "malwarebazaar"`) and is never shown again in the admin UI. Turning the feed off deletes the key and the saved list; existing flags stay.

#### Rules

Rules support every Santa policy, including `CEL` (with a CEL expression), `SILENT_GUI_BLOCKLIST` and `SILENT_TTY_BLOCKLIST`. They also support the optional custom URL, event detail button label and notification app name fields.

Deleting a rule, or unassigning a device from a managed app or machine rule, keeps a tombstone in the `rules` collection. Affected devices then receive a `REMOVE` for that rule on their next sync, without needing a clean sync. Saving a rule with the same identifier again revives it.

#### Sync protocol

The sync endpoints use Santa's official protocol definitions (`santa.sync.v1`). They accept JSON or binary protobuf (Santa's `SyncEnableProtoTransfer`) and reply in the same format. Request bodies can be uncompressed, `gzip` or `deflate`. File access events and rules users create in Standalone mode are stored and shown on the **File Access** and **Standalone** pages.

#### Metrics

Santa can send its metrics to the metrics service, which writes them to the log. Set `MetricURL` to `https://<host>/metrics` and `MetricFormat` to `rawjson` or `monarchjson` in Santa's configuration profile. The `proto` format isn't supported: Santa sends it through the sync service to a different URL.

Santa sends these uploads without the `X-API-Key` header (it doesn't apply `SyncExtraHeaders` to them), so the endpoint can't tell who sent them. It only accepts bodies up to 1 MiB in one of the two formats and rejects anything else, so random data never reaches the logs.

### Current status

The sync server is feature complete for Santa's `santa.sync.v1` protocol:

* Every rule type and policy, including CEL rules, and removal of deleted rules without a clean sync
* Per-device modes, settings and one-off clean syncs, set from the admin UI
* Event, file access event and Standalone rule uploads, browsable in the admin UI
* Self-service allow requests, where a rule becomes global once enough users have asked for it
* Optional MalwareBazaar checks that flag known malware for review
* Metrics uploads, a local Docker Compose setup, and example Terraform for GKE, AKS and EKS

Not there yet:

* Sign-in with identity providers other than Microsoft Entra ID
* Managing File Access Authorization rules. Santa's open-source sync protocol can't deliver them, so they still have to come from Santa's configuration profile

### What's with the stupid name

Well, keeping in the tradition of giving weird and funny name to things in the Mac community, who would tell Santa what to do? Mrs. Santa was the only correct answer!