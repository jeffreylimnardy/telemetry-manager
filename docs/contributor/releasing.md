# Releasing

## Release Process

This release process covers the steps to release new major and minor versions for the Kyma Telemetry module.

Together with the module release, prepare a new release of the [opentelemetry-collector-components](https://github.com/kyma-project/opentelemetry-collector-components). You will need this later in the release process of the Telemetry Manager. The version of `opentelemetry-collector-components` will include the Telemetry Manager version as part of its version (`{CURRENT_OCC_VERSION}-{TELEMETRY_MANAGER_VERSION}`).

1. Verify that all issues in the [GitHub milestone](https://github.com/kyma-project/telemetry-manager/milestones) related to the version are closed.

2. Close the milestone.

3. Create a new [GitHub milestone](https://github.com/kyma-project/telemetry-manager/milestones) for the next version.

4. In the `telemetry-manager` repository, create a release branch.
   The name of this branch must follow the `release-x.y` pattern, such as `release-1.0`.

   ```bash
   git fetch upstream
   git checkout --no-track -b {RELEASE_BRANCH} upstream/main
   git push upstream {RELEASE_BRANCH}
   ```

5. Bump the `telemetry-manager/{RELEASE_BRANCH}` branch with the new versions for the dependent images.
   Create a PR to `telemetry-manager/{RELEASE_BRANCH}` with the following changes:
   - Update Docker images in the `.env` file:
      - Update the `ENV_IMAGE` variable, update the tag of the `telemetry-manager` image with the new module version following the `x.y.z` pattern. For example, `ENV_IMG=europe-docker.pkg.dev/kyma-project/prod/telemetry-manager:1.0.0`.
      - Update the `ENV_OTEL_COLLECTOR_IMAGE` variable, and update the tag of the `kyma-otel-collector` image with the new version released from the [opentelemetry-collector-components](https://github.com/kyma-project/opentelemetry-collector-components) repository. For example, `ENV_OTEL_COLLECTOR_IMAGE=europe-docker.pkg.dev/kyma-project/prod/kyma-otel-collector:0.100.0-1.0.0`.
   - Update `config/manager/kustomization.yaml`:
      - Update the `newTag` field for the `telemetry-manager` image with the new module version following the `x.y.z` pattern, such as `1.0.0`.
   - `make generate`
      - Run `make generate` to update the images in the `sec-scanners-config.yaml` and other files.

6. Merge the PR.

7. To make sure that the release tags point to the HEAD commit of the `telemetry-manager/{RELEASE_BRANCH}` branch, rebase the upstream branch into the local branch after the merge was successful.

   ```bash
   git fetch upstream
   git rebase upstream/{RELEASE_BRANCH} {RELEASE_BRANCH}
   ```

8. In the `telemetry-manager/{RELEASE_BRANCH}` branch, create release tags for the HEAD commit.

   ```bash
   git tag {RELEASE_VERSION}
   ```

   Replace {RELEASE_VERSION} with the new module version, for example, `1.0.0`.

9.  Push the tags to the upstream repository.

   ```bash
   git push upstream {RELEASE_VERSION}
   ```

   The {RELEASE_VERSION} tag triggers the GitHub actions `Build Image` and `Tag Release`. The `Build Image` action builds the `telemetry-manager` image, tags it with the module version, and pushes it to the production registry. `Tag Release` action creates the GitHub release.

10. Verify the [status](https://github.com/kyma-project/telemetry-manager/actions/workflows/build-image.yml) of the `Build Image` GitHub action and the [status](https://github.com/kyma-project/telemetry-manager/actions/workflows/tag-release.yml) of the `Tag Release` GitHub action.
   - Once the `Build Image` and the `Tag Release` GitHub actions succeed, the new GitHub release is available under [releases](https://github.com/kyma-project/telemetry-manager/releases).
   - If the `Build Image` or the `Tag Release` GitHub action fails, re-trigger them by removing the {RELEASE_VERSION} tag from upstream and pushing it again:

     ```bash
     git push --delete upstream {RELEASE_VERSION}
     git push upstream {RELEASE_VERSION}
     ```

11. If the previous release was a bugfix version (patch release) that contains cherry-picked changes, these changes might appear again in the generated change log. If there are redundant entries, edit the release description and remove them.

## Changelog

Every PR's title must adhere to the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification for an automatic changelog generation. It is enforced by a [semantic-pull-request](https://github.com/marketplace/actions/semantic-pull-request) GitHub Action.

### Pull Request Title

Due to the Squash merge GitHub Workflow, each PR results in a single commit after merging into the main development branch. The PR's title becomes the commit message and must adhere to the template:

`type(scope?): subject`

#### Type

- **feat**. A new feature or functionality change.
- **fix**. A bug or regression fix.
- **docs**. Changes regarding the documentation.
- **test**. The test suite alternations.
- **deps**. The changes in the external dependencies.
- **chore**. Anything not covered by the above categories (e.g., refactoring or artefacts building alternations).

Beware that PRs of type `chore` do not appear in the Changelog for the release. Therefore, exclude maintenance changes that are not interesting to consumers of the project by marking them with chore type:

- Dotfile changes (.gitignore, .github, and so forth).
- Changes to development-only dependencies.
- Minor code style changes.
- Formatting changes in documentation.

#### Subject

The subject must describe the change and follow the recommendations:

- Describe a change using the [imperative mood](https://en.wikipedia.org/wiki/Imperative_mood).  It must start with a present-tense verb, for example (but not limited to) Add, Document, Fix, Deprecate.
- Start with an uppercase, and not finish with a full stop.
- Kyma [capitalization](https://github.com/kyma-project/community/blob/main/docs/guidelines/content-guidelines/02-style-and-terminology.md#capitalization) and [terminology](https://github.com/kyma-project/community/blob/main/docs/guidelines/content-guidelines/02-style-and-terminology.md#terminology) guides.
