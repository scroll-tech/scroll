### Purpose or design rationale of this PR

*Describe your change. Make sure to answer these three questions: What does this PR do? Why does it do it? How does it do it?*


### PR title

Your PR title must follow [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary) (as we are doing squash merge for each PR), so it must start with one of the following [types](https://github.com/angular/angular/blob/22b96b9/CONTRIBUTING.md#type):

- [ ] build: Changes that affect the build system or external dependencies (example scopes: yarn, eslint, typescript)
- [ ] ci: Changes to our CI configuration files and scripts (example scopes: vercel, github, cypress)
- [ ] docs: Documentation-only changes
- [ ] feat: A new feature
- [ ] fix: A bug fix
- [ ] perf: A code change that improves performance
- [ ] refactor: A code change that doesn't fix a bug, or add a feature, or improves performance
- [ ] style: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- [ ] test: Adding missing tests or correcting existing tests


### Deployment tag versioning

Has `tag` in `common/version.go` been updated or have you added `bump-version` label to this PR?

- [ ] No, this PR doesn't involve a new deployment, git tag, docker image tag
- [ ] Yes


### Breaking change label

Does this PR have the `breaking-change` label?

- [ ] No, this PR is not a breaking change
- [ ] Yes
