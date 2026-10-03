# Inotia2 local gold shop, installation version 1013

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

The first eleven products above retain their order and prices. These thirteen
products follow them, for 24 total; no general item-database enumeration is used.

| Added item | Gold per unit |
|---|---:|
| 라시엘의 검 | 50,000 |
| 라시엘의 활 | 50,000 |
| 선택받은 자의 관 | 30,000 |
| 선택받은 자의 뿔투구 | 30,000 |
| 축복받은 부활주문서 | 5,000 |
| 봉인된 용사의 인장 | 10,000 |
| 봉인된 각성의 인장 | 10,000 |
| 무한의 가방 | 30,000 |
| 카오스세트 | 50,000 |
| 에픽카오스세트 | 100,000 |
| 강화세트 | 30,000 |
| 공성전세트 | 30,000 |
| 부활의 기도문 | 10,000 |

Equipment keeps its native level and class restrictions. Packs keep their native
contents and activation rules; names and server-era prices are not reconstructed.

Use the game's normal merchant controls: choose a product, press OK, choose
the quantity when offered, then confirm. CLR closes the window. Existing
item descriptions, item effects and consumable-use rules remain native.
No standalone guild/chat/storage service products are added. The native siege
pack does contain online-only goods such as a guild megaphone; the corresponding
retired online services are not restored by this shop patch.

The native purchase callback still clones a stock item, checks bag/stack
capacity, inserts it successfully and only then debits gold. Insufficient
gold, no capacity and cancellation leave inventory and gold unchanged.
Stock templates are released by the native merchant cleanup. Ordinary shops
use their original initializer and original pricing, including noncatalog
items in the local shop's sell/buyback view.

The emulator, core, frontend and audio/frame patches use the exact v1008
baseline. The latest checkout supplies only the local-shop compatibility
overlay. Package ID and the existing Nightly signing certificate stay the
same; versionCode 1013 can update version 1012 without deleting app data.
The game ZIP, persistent-save identity, original BSS bootstrap size and full
state geometry are unchanged. Restoring a v1008 state installs the guarded shop hooks. A complete v1012
state upgrades to the current helper/catalog after all original, legacy and
current spans are checked; mixed or corrupted installations are rejected
before any writes. Reopen a shop restored while already open to refresh stock.
New full states include both code and shop scope.

Verification includes portable/JIT and ARM64 native execution of the new
Thumb helpers, title-hash gating, atomic span validation, idempotence,
ordinary-merchant fallback, stock initialization and scope cleanup. Private
replay against the user's authorized game/save verified a purchased seal,
the exact gold debit, insufficient-gold message 6, full-bag message 7 and exit.
Game packages, private saves, memory dumps and game screenshots stay outside
the repository and public workflow artifacts.

Private release replay verified all thirteen added products, one-unit quantities
and each exact gold debit, all four native packs opening into their client-defined
contents, the 24-entry stock grid, v1008 and v1012 full-state restore/reopen, and
lossless WFS restore with the existing game ZIP/save hashes. Android APK payload,
Nightly certificate, installation version 1013 and 16 KiB ELF alignment were
verified. Required Linux, Android and ARM64 jobs passed for code commit 9d5d904.
