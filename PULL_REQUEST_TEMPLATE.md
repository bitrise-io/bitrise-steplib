### What to do if the build fails?

At the moment contributors do not have access to the CI workflow triggered by StepLib PRs. In case of a failed build, we ask for your patience. Maintainers of Bitrise Steplib will sort it out for you or inform you if any further action is needed.

### New Pull Request Checklist

*Please mark the points which you did / accept.*

- [ ] __I will not move an already shared step version's tag to another commit__
- [ ] I read the [Step Development Guideline](https://github.com/bitrise-io/bitrise/blob/master/_docs/step-development-guideline.md)
- [ ] I have a test for my Step, which can be run with `bitrise run test` (in the step's repository)
- [ ] I did run `bitrise run audit-this-step` (in the step's repository - note, if you don't have this workflow in your `bitrise.yml`, [you can copy it from the step template](https://github.com/bitrise-steplib/step-template/blob/master/bitrise.yml).)
- [ ] I read and accept the [Abandoned Step policy](https://github.com/bitrise-io/bitrise-steplib#abandoned-step-policy)
