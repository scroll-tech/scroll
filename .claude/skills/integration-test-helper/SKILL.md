---
name: integration-test-helper
description: Helps launching the full process of integration test, also investigate and report the results.
---

## Target directory

The whole process should be run under current directory, unless it is specified to ($ARGUMENTS[0])
Under the target dir there are the stuff and instructions.

## Instructions

First read `README.md` under target directory, instructions should be under heading named ($ARGUMENTS[1]). If there is no such a heading name, just try the "Test" heading.

## Run each step listed in instructions

The instructions often contain multiple steps which should be completed in sequence. Following are some rules MUST be obey while handling each step:

### "Must do" while executing commands in steps

Any command mentioned in steps should be executed by Bash tool, with following MUST DO for handling the outputs:

+ Redirect the output of bash tool, both from stdout andstderr, into a local log file for investigating later. The file name should be in format as `<desc_of_ccommand>_<day>_<time>.log`
+ Do not read the whole log file. Just investigate the last 50 lines (use "tail -n 50") for possible error message.

It may need to jump to other directories for executing a step. We MUST go back to target directory after every step has been completed. Also, DO NOT change anything outside of target directy.

### When error raised
Command execution should get success return. If error raised while executing, do following process:

1. Try to analysis the reason of error, first from the caught error message. If there is no enough data, grep useful information from the log file of whole output just captured.

2. MUST ASK USER for next action, options are:
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

