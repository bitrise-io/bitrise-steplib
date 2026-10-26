### Pull Request Checklist

- [ ] I have read the [Step Development Guideline](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step)
- [ ] I have tested the step in an actual CI/CD workflow (preferably as a workflow in the step repo's `bitrise.yml`)
- [ ] I did run `bitrise run audit-this-step` (in the step's repository - note, if you don't have this workflow in your `bitrise.yml`, [you can copy it from the step template](https://github.com/bitrise-steplib/step-template/blob/master/bitrise.yml).)
- [ ] I am aware of the [Abandoned Step policy](https://github.com/bitrise-io/bitrise-steplib#abandoned-step-policy)
- [ ] __I will not move an already shared step version's tag to another commit__

### What's next

After your PR is opened:

- Make sure you accept the Contributor terms. @CLAassistant is going to check this and tell you what to do.
- A Bitrise engineer is going to take a look at the PR. Brand new step submissions get reviewed in more detail, updates to existing steps are usually approved instantly.
- CI checks: At the moment, contributors do not have access to the CI workflow triggered by PRs. In case of a failed build, we ask for your patience.


