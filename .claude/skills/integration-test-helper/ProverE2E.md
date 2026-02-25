## Notes for handling ProverE2E

+ Ensure the `conf` dir has been correctly linked and remind user which path it currently links to.

+ If some files are instructed to be generated, but they have been existed, NEVER refer the content before the generation. They may be left from different setup and contain wrong message for current process.

+ In step 4, if the `l2.validium_mode` is set to true, MUST Ask User for decryption key to fill the `sequencer.decryption_key` field. The key must be a hex string WITHOUT "0x" prefix.