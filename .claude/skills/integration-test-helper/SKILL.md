---
name: integration-test-helper
description: Assist with the process described in the specified directory to prepare or advance integration tests. The target directory and instruction section can be specified, like "tests/prover-e2e test".
model: sonnet
allowed-tools: Bash(make *), Bash(tee *), Bash(jq *)
---

This skill helps launching the full process described in the instructions, also investigate and report the results.

## Target directory

The **target directory** under which the setup process being run is: $ARGUMENTS[0].
Under the target dir there are the stuff and instructions. If the target dir above is empty, just use !`pwd`.

## Instructions

First read `README.md` under target directory, instructions should be under heading named ($ARGUMENTS[1]). If there is no such a heading name, just try the "Test" heading.

In additional, there are two optional places for more knowledge about current instructions:

+ An .md file under current skill dir, named from the top header of the `README.md` file or the name of target directory.
  For example, if the target dir is `tests/prover-e2e`, the top header in `README.md` has "ProverE2E", so there may be a .md file named as `prover-e2e.md` or `ProverE2E.md`

+ All files under `experience` path (if it existed) of target dir contains additional experience, which is specialized for current host

## Run each step listed in instructions

The instructions often contain multiple steps which should be completed in sequence. Following are some rules MUST be obey while handling each step:

### "Must do" while executing commands in steps

Any command mentioned in steps should be executed by Bash tool, with following MUST DO for handling the outputs:

+ Use "| tee <log_file>" to capture output of bash tool into local file for investigating later. The file name of log should be in format as `<desc_of_ccommand>_<day>_<time>.log`
+ Do not read all output, after "| tee", use "|tail -n 50" to only catch the possible error message. That should be enough for common case.

It may need to jump to other directories for executing a step. We MUST go back to target directory after every step has been completed. Also, DO NOT change anything outside of target directy.

### When error raised
Command execution should get success return. If error raised while executing, do following process:

1. Try to analysis the reason of error, first from the caught error message. If there is no enough data, grep useful information from the log file of whole output just captured.

2. Ask User for next action, options are:
   + Retry with resolution derived from error analyst
   + Retry, with user provide tips to resolve the issue
   + Just retry, user has resolved the issue by theirself
   + Stop here, discard current and following steps, do after completion

Error often caused by some mismacthing of configruation in current host. Here are some tips which may help:

* Install the missing tools / libs via packet manager
* Fix the typo, or complete missed fields in configuration files
* Copy missed files, it may be just put in some place of the project or can be downloaded according to some documents.

## After completion

When every step has done, or the process stop by user, make following materials before stop:

+ Package all log files generated before into a tarball and save it in tempoaray path. Then clear all log files.
+ Generate a report file under target directory, with file name like `report_<day>_<time>.txt`.
+ For steps once failed and being resolved later, record the resolution into a file under `experience` path in target dir.
