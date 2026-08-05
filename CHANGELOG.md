## v0.1.2 (August 5, 2026)

* **Breaking**: TOML config keys now use hyphens (`app-port`, `allowed-users`) instead of underscores (`app_port`, `allowed_users`)
* Consolidate flag definitions and source resolution into `internal/cmd/wh/args` package
* Remove `urfave/cli-altsrc` vendored dependency

## v0.1.1 (July 30, 2026)

* Add support for setting default parameters with layerable TOML configuration files
* Default configuration file is `/etc/wormhole/cli.toml`

# Changelog

## v0.1.0 (July 23, 2026)

* Initial open source release

