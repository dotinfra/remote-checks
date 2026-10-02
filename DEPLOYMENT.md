# Deployment — Scaleway Serverless Containers

The service is packaged as a `FROM scratch` Docker image (~6 MB, static Go binary) and deployed to [Scaleway Serverless Containers](https://www.scaleway.com/en/serverless-containers/).

## Architecture

- **Registry**: GHCR (`ghcr.io/dotinfra/remote-checks`), public image, tagged with the commit SHA and `latest`.
- **Runtime**: Scaleway Serverless Container, region `fr-par` by default, port `8080`, HTTP health probe on `/healthz`.
- **Delivery**: GitHub Actions workflow `.github/workflows/deploy.yml` — on every push to `master`, after tests pass, it builds and pushes the image to GHCR, points the Scaleway container at the new image, triggers a deploy, and smoke-tests `https://<container-domain>/healthz`.

## One-time setup

The deploy workflow assumes the container already exists in Scaleway. Create it once:

1. **Create a Serverless Container namespace** (console: Serverless > Containers, or API):

   ```sh
   curl -X POST \
     -H "X-Auth-Token: $SCW_SECRET_KEY" \
     -H "Content-Type: application/json" \
     -d '{"name": "remote-checks", "project_id": "<SCW_PROJECT_ID>"}' \
     https://api.scaleway.com/containers/v1/regions/fr-par/namespaces
   ```

2. **Create the container** in that namespace, pointing at the GHCR image and port 8080:

   ```sh
   curl -X POST \
     -H "X-Auth-Token: $SCW_SECRET_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "remote-checks",
       "namespace_id": "<NAMESPACE_ID>",
       "registry_image": "ghcr.io/dotinfra/remote-checks:latest",
       "port": 8080,
       "memory_limit": 128,
       "min_scale": 0,
       "max_scale": 5,
       "health_check": {"http": {"path": "/healthz"}}
     }' \
     https://api.scaleway.com/containers/v1/regions/fr-par/containers
   ```

   Recommended settings: `min_scale: 0` (scale to zero — cold starts are ~ms for this binary), `memory_limit: 128` MB (CPU is derived from memory; the service is I/O-bound, 128 MB is ample), `max_scale: 5`.

3. **Add the GitHub repository secrets** (Settings > Secrets and variables > Actions):

   | Secret             | Value                                          |
   |--------------------|------------------------------------------------|
   | `SCW_SECRET_KEY`   | Scaleway API secret key (IAM credential)        |
   | `SCW_CONTAINER_ID` | UUID of the container created in step 2        |

   Optional repository variable: `SCW_REGION` (default `fr-par`).

   The API key needs the `ContainersReadWrite` permission set (or an equivalent policy on the container namespace).

4. **Create a GitHub environment** named `production` (Settings > Environments) if you want a manual approval gate before deployment; the workflow's `deploy` job references it.

5. **First deploy**: push to `master`, or run the workflow manually (Actions > Deploy to Scaleway > Run workflow).

## Deployed endpoints

The container gets an auto-generated domain (`<name>.<namespace>.fnc.fr-par.scw.cloud`):

- `https://<domain>/` → 302 to dotinfra.fr
- `https://<domain>/healthz` → 200
- `https://<domain>/checks?url=<target>[&proxy=<proxy-url>]` → JSON check report

## Notes

- The container is `public` by default (anyone with the URL can invoke it). Set `privacy: private` on the container if you want Scaleway-auth-only invocation.
- `PORT` env var is respected by the binary but Scaleway injects its own port handling; keep the container `port` at `8080` (the binary's default).
- Container logs are available in the Scaleway Cockpit (Grafana): `{resource_type="serverless_container"}`.
