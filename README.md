# New Bitrise StepLib

You can find the collection of all [Bitrise integrations](https://www.bitrise.io/integrations) in this repository under [/steps](https://github.com/bitrise-io/bitrise-steplib/tree/master/steps).

## Contribution

If you find something missing from the steps, you can drop us an issue, or [create your own step](http://devcenter.bitrise.io/docs/step-dev). See our example for creating & sharing a new step under [/step-template](https://github.com/bitrise-steplib/step-template).

### Install Bitrise CLI

Install the [Bitrise CLI](https://www.bitrise.io/cli) to run `bitrise` on your machine locally.

You can install it via Homebrew:

`brew update && brew install bitrise`

Or check the latest release with instructions at: [https://github.com/bitrise-io/bitrise/releases](https://github.com/bitrise-io/bitrise/releases)

### Share your step

After implementing your own step, you can share it with other Bitrisers using this StepLib via [`stepman`](https://github.com/bitrise-io/stepman).

If you are ready, just run `stepman share` and follow the instructions.

Follow [@bitrise](https://twitter.com/bitrise) on Twitter for #status and step updates 🚀.


## Abandoned Step policy

We try to keep this Step Library up-to-date and active. Steps shared in this collection have to be actively maintained to receive fixes / updates when required (e.g. security issue fixes or general usability fixes).

**If you're a Step maintainer** you're not required to accept every Pull Request sent to your Step **but you should be reachable within a reasonable timeframe**. If we try to contact you several times regarding an important fix/update in your Step and you refuse to answer for several weeks *we might deprecate, remove or replace your Step* in the collection. Abandoned Steps can be a threat for those who use it, please keep this in mind if you decide to share your Step with others!

**If you shared a Step but you're no longer able to or you don't want to maintain it** please create a GitHub issue in this repository (https://github.com/bitrise-io/bitrise-steplib).

**If you're a user of a Step which has critical (security or functionality) issues** please create a ticket in the Step's Issue Tracker (every Step declares the preferred way of reporting issues with the `support_url` attribute - [see](https://github.com/bitrise-io/bitrise-steplib/blob/master/steps/activate-ssh-key/3.1.0/step.yml#L15)) first. If you don't get a response from the Step's maintainer for an extended period (reasonably, in general, for more than a couple of weeks) please create a GitHub issue in this repository (https://github.com/bitrise-io/bitrise-steplib) and we'll try to resolve the issue, following the Abandoned Step policy. *Please be patient* and keep in mind that everyone who contribute to this collection does that with an intention to help You by providing a Step for you to use, don't be rude to Step maintainers!
