
Important information about the VST3 version
--------------------------------------------

During installation, the VST3 version is not enabled by default, and we highly recommend using VST2 for the time being. If you need to install VST3, e.g. if you have existing projects that already include VST3 versions of our software, please use the customization option in the installer, and enable VST3.

After working on full VST3 versions for several months, we were still encountering random errors and glitches - we had simply underestimated the complexity. Steinberg's advice was to do a complete rewrite as "simplified" VST3 (also included in SDK). However, rewriting would have meant considerable work, so we finally decided to call our current VST3 implementation "experimental" until we can afford to take the necessary steps.

Known issues with the VST3 versions:

- they don't report parameters on preset changes
- in Wavelab, they show symbols instead of registration info
- they crash FL 11 beta
- they don't show the selected patch name when a project is reopened

Again: We recommend using the VST2 versions for now. They behave the same, they sound the same, they're tried and tested!