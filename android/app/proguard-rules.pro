# App-specific R8 rules.
#
# The ebitenmobile-generated AAR contributes consumer rules that retain the Go
# binding packages used through JNI. Android Gradle Plugin and the Google Ads
# dependencies contribute their own rules for manifest components and SDK
# entry points, so the app itself does not need broad -keep rules here. Keeping
# this file intentionally narrow lets R8 shrink, optimize, and obfuscate the
# remaining DEX code for the Play technical-quality thresholds.
