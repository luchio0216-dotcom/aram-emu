# Inotia2 acquisition identification, installation version 1014

Newly acquired unidentified equipment is identified immediately during native
inventory registration, including field proximity pickup, manual pickup and
merchant purchases. Shop stock remains unidentified until a purchase succeeds.
Use the native scroll action rather than changing item definitions or replacing
generated equipment. Identification does not consume a scroll or debit gold;
the original merchant transaction still debits its original price.

The acquisition hook runs after the native bag-capacity check and before stack
insertion can merge or free the incoming item. The scroll action's native
equipment predicate leaves bags, stackable consumables and already identified
equipment unchanged. The helper refreshes the cached item ID if identification
resolves the native starter dagger, retains the native quantity return, reloads
the selected destination slot and resumes original insertion/notifications.
UIDs, native generated options, level/class requirements and item effects stay
native. Existing unidentified equipment already in a bag is not scanned.

Cancelled purchases, insufficient gold and full bags do not reach the hook.
Full-bag field drops remain on the ground. Inventory discard protection and
reapproach pickup remain at the v1008 baseline. There are no added frame polls,
audio hooks, input injection, or per-instruction callbacks.

The exact post-autoloot client hash gates the two expected-original memory
spans. Both are validated before writes; complete original or current images
are accepted and mixed/corrupted installations are rejected. Existing v1008,
v1012 and v1013 full states have the original acquisition spans and receive the
hook on restoration. Mapping geometry, game ZIP and persistent-save hashes
remain unchanged. The existing guarded 24-product shop is unchanged.

All runtime dependencies and performance/input/save/CALL patches retain the
pinned v1008 release baseline. Dependency and distribution licensing remain
as reviewed for that unchanged baseline. The Nightly package/signing identity
is unchanged; versionCode 1014 updates 1013 without deleting app data.

Portable/JIT and ARM64 native tests use authored synthetic callees to verify
call order, live object ownership, cached ID refresh, displaced instructions,
stack/callee preservation, title gating, idempotence and atomic validation.
Private replay with authorized game/save data checks native purchase and field
pickup identification, option/UID preservation and failed-capacity behavior.
Game files, saves, dumps and screenshots never enter source or public CI.
