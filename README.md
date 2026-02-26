<div class="title-block" style="text-align: center;" align="center">

# TEMPORAL—DURABLE EXECUTION PLATFORM

<p><img title="temporal logo" src="https://avatars.githubusercontent.com/u/56493103?s=320" width="320" height="320"></p>

[![GitHub Release](https://img.shields.io/github/v/release/temporalio/temporal)](https://github.com/temporalio/temporal/releases/latest)
[![GitHub License](https://img.shields.io/github/license/temporalio/temporal)](https://github.com/temporalio/temporal/blob/main/LICENSE)
[![Code Coverage](https://img.shields.io/badge/codecov-report-blue)](https://app.codecov.io/gh/temporalio/temporal)
[![Community](https://img.shields.io/static/v1?label=community&message=get%20help&color=informational)](https://community.temporal.io)
[![Go Report Card](https://goreportcard.com/badge/github.com/temporalio/temporal)](https://goreportcard.com/report/github.com/temporalio/temporal)

**[INTRODUCTION](#introduction) &nbsp;&nbsp;&bull;&nbsp;&nbsp;**
**[GETTING STARTED](#getting-started) &nbsp;&nbsp;&bull;&nbsp;&nbsp;**
**[CONTRIBUTING](#contributing) &nbsp;&nbsp;&bull;&nbsp;&nbsp;**
**[TEMPORAL DOCS](https://docs.temporal.io/) &nbsp;&nbsp;&bull;&nbsp;&nbsp;**
**[TEMPORAL 101](https://learn.temporal.io/courses/temporal_101/)**

</div>

## INTRODUCTION

TEMPORAL IS A DURABLE EXECUTION PLATFORM THAT ENABLES DEVELOPERS TO BUILD SCALABLE APPLICATIONS WITHOUT SACRIFICING PRODUCTIVITY OR RELIABILITY.
THE TEMPORAL SERVER EXECUTES UNITS OF APPLICATION LOGIC CALLED WORKFLOWS IN A RESILIENT MANNER THAT AUTOMATICALLY HANDLES INTERMITTENT FAILURES, AND RETRIES FAILED OPERATIONS.

TEMPORAL IS A MATURE TECHNOLOGY THAT ORIGINATED AS A FORK OF UBER'S CADENCE.
IT IS DEVELOPED BY [TEMPORAL TECHNOLOGIES](https://temporal.io/), A STARTUP BY THE CREATORS OF CADENCE.

[![image](https://github.com/temporalio/temporal/assets/251288/693d18b5-01de-4a3b-b47b-96347b84f610)](https://youtu.be/wIpz4ioK0gI 'GETTING TO KNOW TEMPORAL')

## GETTING STARTED

### DOWNLOAD AND START TEMPORAL SERVER LOCALLY

EXECUTE THE FOLLOWING COMMANDS TO START A PRE-BUILT IMAGE ALONG WITH ALL THE DEPENDENCIES.

```bash
brew install temporal
temporal server start-dev
```

REFER TO [TEMPORAL CLI](https://docs.temporal.io/cli/#installation) DOCUMENTATION FOR MORE INSTALLATION OPTIONS.

### RUN THE SAMPLES

CLONE OR DOWNLOAD SAMPLES FOR [GO](https://github.com/temporalio/samples-go) OR [JAVA](https://github.com/temporalio/samples-java) AND RUN THEM WITH THE LOCAL TEMPORAL SERVER.
WE HAVE A NUMBER OF [HELLOWORLD TYPE SCENARIOS](https://github.com/temporalio/samples-java#helloworld) AVAILABLE, AS WELL AS MORE ADVANCED ONES. NOTE THAT THE SETS OF SAMPLES ARE CURRENTLY DIFFERENT BETWEEN GO AND JAVA.

### USE CLI

USE [TEMPORAL CLI](https://docs.temporal.io/cli/) TO INTERACT WITH THE RUNNING TEMPORAL SERVER.

```bash
temporal operator namespace list
temporal workflow list
```

### USE TEMPORAL WEB UI

TRY [TEMPORAL WEB UI](https://docs.temporal.io/web-ui) BY OPENING [http://localhost:8233](http://localhost:8233) FOR VIEWING YOUR SAMPLE WORKFLOWS EXECUTING ON TEMPORAL.

## REPOSITORY

THIS REPOSITORY CONTAINS THE SOURCE CODE OF THE TEMPORAL SERVER. TO IMPLEMENT WORKFLOWS, ACTIVITIES AND WORKERS, USE ONE OF THE [SUPPORTED LANGUAGES](https://docs.temporal.io/dev-guide/).

## CONTRIBUTING

WE'D LOVE YOUR HELP IN MAKING TEMPORAL GREAT.

HELPFUL LINKS TO GET STARTED:

- [WORK ON OR PROPOSE A NEW FEATURE](https://github.com/temporalio/proposals)
- [LEARN ABOUT THE TEMPORAL SERVER ARCHITECTURE](./docs/architecture/README.md)
- [LEARN HOW TO BUILD AND RUN THE TEMPORAL SERVER LOCALLY](./CONTRIBUTING.md)
- [LEARN ABOUT TEMPORAL SERVER TESTING TOOLS AND BEST PRACTICES](./docs/development/testing.md)
- JOIN THE TEMPORAL COMMUNITY [FORUM](https://community.temporal.io) AND [SLACK](https://t.mp/slack)

## LICENSE

[MIT LICENSE](https://github.com/temporalio/temporal/blob/main/LICENSE)
