# Inotia2 local gold shop, installation version 1012

Selecting the system menu's cash-item purchase entry opens a local merchant
window immediately. It makes no network connection and uses in-game gold.
This is a new local catalog with deliberately assigned unit prices; it does
not reconstruct the retired server's catalog, currency or purchase history.

| Item | Gold per unit |
|---|---:|
| 축복받은 용사의 인장 | 10,000 |
| 각성의 인장 | 10,000 |
| 정화의 보석 | 5,000 |
| 봉인된 에테르 | 10,000 |
| 봉인된 스킬북 | 10,000 |
| 초월의 영약(힘) | 20,000 |
| 초월의 영약(민첩) | 20,000 |
| 초월의 영약(체력) | 20,000 |
| 초월의 영약(지능) | 20,000 |
| 초월의 영약(정신) | 20,000 |
| 신수의 물약 | 20,000 |

Use the game's normal merchant controls: choose a product, press OK, choose
the quantity when offered, then confirm. CLR closes the window. Existing
item descriptions, item effects and consumable-use rules remain native.
Network-dependent guild/chat/storage services are not represented as goods.

The native purchase callback still clones a stock item, checks bag/stack
capacity, inserts it successfully and only then debits gold. Insufficient
gold, no capacity and cancellation leave inventory and gold unchanged.
Stock templates are released by the native merchant cleanup. Ordinary shops
use their original initializer and original pricing, including noncatalog
items in the local shop's sell/buyback view.

The emulator, core, frontend and audio/frame patches use the exact v1008
baseline. The latest checkout supplies only the local-shop compatibility
overlay. Package ID and the existing Nightly signing certificate stay the
same; versionCode 1012 can update version 1011 without deleting app data.
The game ZIP, persistent-save identity, original BSS bootstrap size and full
state geometry are unchanged. Restoring a v1008 full machine state reinstalls
the guarded shop hooks, and new full states include both code and shop scope.

Verification includes portable/JIT and ARM64 native execution of the new
Thumb helpers, title-hash gating, atomic span validation, idempotence,
ordinary-merchant fallback, stock initialization and scope cleanup. Private
replay against the user's authorized game/save verified a purchased seal,
the exact gold debit, insufficient-gold message 6, full-bag message 7 and exit.
Game packages, private saves, memory dumps and game screenshots stay outside
the repository and public workflow artifacts.
