<img src="web/public/icon.v1.png" height="64" />

# GeoDuels

A free multiplayer geography game with community maps and competitive duels. Play at [geoduels.io](https://geoduels.io/).

Not a pretty readme, but does the job.

## Local development

Use Docker, Go 1.26, and Node 22 (matching CI).

```sh
cp infra/compose/.env.example infra/compose/.env
cp web/.env.local.example web/.env.local
./infra/scripts/dev-up.sh
npm --prefix web ci
npm --prefix web run dev
```

Open `http://localhost:3000`. Start optional workers with `./infra/scripts/compose.sh up -d moderation-worker discord-worker`; stop with `./infra/scripts/compose.sh down`.

## Development references

- [AGENTS.md](AGENTS.md): constraints and cross-cutting behavior to preserve.
- [Development notes](docs/development.md): generated code, meaningful verification, local infrastructure, releases, and map tools.
- [Extension notes](extension/README.md): local installation and production packaging.
- [Contributor agreement](CONTRIBUTOR_LICENSE_AGREEMENT.md) and [license](LICENSE): contribution and licensing terms.
