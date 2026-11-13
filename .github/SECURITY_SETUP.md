# Security alerts and checks

- **Dependabot**: alerts and grouped security fix PRs for vulnerable Go dependencies.
  Routine Go version update PRs are disabled. Weekly grouped GitHub Actions updates
  target `dev` to keep CI maintained. Nothing is merged automatically.
- **CodeQL**: Go source analysis with `security-extended` queries on PRs into `dev`
  and `stable`, pushes to those branches, and weekly on the default branch.
  Findings appear under **Security > Code scanning**.
- **Security workflow**: `govulncheck` on PRs, pushes, and daily on the default
  branch. Reachable dependency and Go standard library vulnerabilities fail the
  check. Findings appear in Actions logs.

## Activate on GitHub

1. Merge these files through `dev` into the default branch, currently `stable`.
   Dependabot configuration and scheduled workflows must exist on that branch.
2. In repository **Settings > Advanced Security** (also called **Code security**
   or **Code security and analysis** in some layouts), enable the dependency
   graph, Dependabot alerts, and Dependabot security updates. Adding the YAML
   configuration alone does not enable these repository settings.
3. Use the committed CodeQL workflow as advanced setup; avoid also enabling CodeQL
   default setup. Code scanning is available for this public repository; making
   it private may require GitHub Code Security licensing.
4. Run **Actions > Security > Run workflow** and **Actions > CodeQL > Run
   workflow** after merging. Verify analysis uploads and inspect the Security tab.
   Existing vulnerabilities can make the Security workflow fail immediately.
5. In personal GitHub **Settings > Notifications**, enable desired Dependabot
   email/web notifications and failed Actions notifications. Repository files
   cannot change anyone's personal notification preferences.
6. Also enable secret scanning and push protection in repository security settings
   to detect supported leaked credentials and block new leaks.

Branch protection is unchanged. Maintainers can require the new checks after
initial findings are fixed.

## Branches and fixes

Dependabot security PRs target the default branch, `stable`. The Go entry has no
`target-branch` because that option prevents its security-update configuration
from applying. CI also runs on PRs into `stable` to test these fixes. Bring merged
security fixes back into `dev` using the usual branch synchronization process.

## Local checks and coverage

```sh
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
CGO_ENABLED=0 govulncheck ./...
```

CI scans on Linux with the latest Go 1.25 patch, matching the Docker build's minor
version. Installing the scanner may download a newer toolchain for the scanner
itself. Update CI and Docker together when upgrading the application toolchain;
Dependabot cannot fix standard library vulnerabilities by updating Go modules.

These checks do not scan Alpine packages or the separate `migrate` executable
installed by the Dockerfile. Container image scanning is a separate check.
Automated scanning also does not replace security review.

References: [Dependabot configuration](https://docs.github.com/en/code-security/concepts/supply-chain-security/about-the-dependabot-yml-file),
[security updates](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/configure-security-updates),
[CodeQL](https://github.com/github/codeql-action),
[govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck).
