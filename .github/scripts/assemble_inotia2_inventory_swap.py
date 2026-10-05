# Authored Thumb wrappers for the known client's inventory UI. Native items
# stay owned by their slots throughout a chain; no game bytes or saves here.
from assemble_inotia2_offline_shop import Asm, bl

CHECK = 0x2ae340
MOVE = 0x2ae420
DRAW = 0x2aef20
FINISH = 0x2aef60
GOT = 0x1924c4

def cmp(a, n, m): a.h(0x4280 | (m << 3) | n)
def add(a, d, n, m): a.h(0x1800 | (m << 6) | (n << 3) | d)

def build():
    patches = []
    a = Asm(CHECK)
    a.h(0xb5f0); a.h(0xb083)  # 32-byte frame, including find-item output
    a.reg(4, 0); a.reg(5, 1)
    a.cmp(4, 0); a.cond(0, 'no'); a.cmp(5, 0); a.cond(0, 'no')
    cmp(a, 4, 5); a.cond(0, 'no')
    # Restrict acceptance to the player's item grid and move mode. Merchant,
    # equipment, bag-header, use and quantity-selection UI keep native rules.
    for off in (0xab8, 0x4ec):
        a.ldr(3, GOT + off); a.load(3, 3); a.byte(3, 3, load=True)
        a.cmp(3, 1); a.cond(1, 'no')
    a.ldr(3, GOT + 0xae8); a.load(7, 3)
    a.half(3, 7, load=True); a.cmp(3, 2); a.cond(1, 'no')
    a.load(3, 7, 4); cmp(a, 3, 4); a.cond(1, 'no')
    a.reg(0, 4); a.bl(0x147eb4)
    a.cmp(0, 0); a.cond(0, 'no')
    a.byte(3, 7, 2, load=True); cmp(a, 0, 3); a.cond(1, 'no')
    # Only a complete stack can be exchanged. A selected partial quantity
    # still follows the game's split/merge transaction instead of losing units.
    a.ldr(3, GOT + 0xad0); a.load(3, 3); a.load(2, 3)
    a.cmp(2, 5); a.cond(8, 'no'); a.h(0x9201)
    a.ldr(3, GOT + 0x5a8); a.load(3, 3); a.load(3, 3)
    a.cmp(3, 0); a.cond(0, 'no')
    a.byte(0, 3, 11, load=True); a.cmp(0, 15); a.cond(8, 'no')
    a.byte(1, 3, 12, load=True); cmp(a, 0, 1); a.cond(2, 'no')
    a.load(3, 3, 16)
    a.ldr(6, GOT + 0x8a8); a.load(6, 6)
    a.h(0x0192); add(a, 2, 2, 6); cmp(a, 2, 3); a.cond(1, 'no')
    a.h(0x0080); add(a, 2, 2, 0); a.load(3, 2)
    cmp(a, 3, 5); a.cond(1, 'no'); a.h(0x9202)
    a.reg(0, 4); a.h(0xa900); a.bl(0x146dd4)
    a.cmp(0, 0); a.cond(0, 'no')
    a.h(0x466b); a.byte(0, 3, load=True); a.h(0x0941)
    # Native bag 5 is a separate item category. Never cross that boundary.
    a.h(0x9a01); a.cmp(1, 5); a.cond(1, 'normal')
    a.cmp(2, 5); a.cond(1, 'no'); a.branch('source')
    a.label('normal'); a.cmp(2, 5); a.cond(0, 'no')
    a.label('source'); a.h(0x0189); add(a, 1, 1, 6)
    a.mov(3, 31); a.h(0x4018); a.h(0x0080); add(a, 1, 1, 0)
    a.h(0x9a02); a.reg(3, 5); a.mov(0, 1); a.branch('done')
    a.label('no'); a.mov(0, 0)
    a.label('done'); a.h(0xb003); a.h(0xbdf0)
    code = a.finish(); assert CHECK + len(code) <= MOVE
    patches.append((CHECK, code))

    a = Asm(MOVE)
    # Preserve original arguments and all callee-saved registers. The native
    # transaction runs first, retaining empty-slot moves and stack merges.
    a.h(0xb5ff); a.h(0xb081); a.bl(0x147464)
    a.cmp(0, 0); a.cond(1, 'done')
    a.h(0x9a03); a.h(0x9b04)
    a.cmp(2, 5); a.cond(8, 'no'); a.cmp(3, 15); a.cond(8, 'no')
    a.h(0x0192); a.h(0x009b); add(a, 2, 2, 3)
    a.ldr(3, GOT + 0x8a8); a.load(3, 3); add(a, 2, 2, 3)
    a.load(1, 2); a.h(0x9801); a.bl(CHECK)
    a.cmp(0, 0); a.cond(0, 'no')
    a.reg(4, 1); a.reg(5, 2); a.reg(6, 3)
    a.reg(0, 6); a.bl(0x147eb4)
    a.cmp(0, 0); a.cond(0, 'no'); a.cmp(0, 99); a.cond(8, 'no')
    # Swap two pointers, leaving the displaced item at the old source slot.
    # Cancel simply releases its overlay; no temporary unowned item exists.
    a.h(0x9b01); a.store(6, 4); a.store(3, 5)
    a.ldr(3, GOT + 0xae8); a.load(3, 3)
    a.store(6, 3, 4); a.byte(0, 3, 2)
    a.mov(0, 2); a.branch('done')
    a.label('no'); a.mov(0, 0)
    a.label('done'); a.h(0xb005); a.h(0xbdf0)
    code = a.finish(); assert MOVE + len(code) <= 0x2ae500
    patches.append((MOVE, code))

    a = Asm(DRAW)
    # Replace only the occupied-target highlight check. The native renderer
    # already applied its category restrictions before reaching this call.
    a.h(0xb537); a.h(0x9c09)  # target at original SP+12, now SP+36
    a.cmp(4, 0); a.cond(0, 'done')
    a.reg(0, 7); a.reg(1, 4); a.bl(CHECK)
    a.cmp(0, 0); a.cond(0, 'done'); a.mov(4, 0)
    a.label('done'); a.reg(3, 4); a.h(0xbc37); a.cmp(3, 0); a.h(0xbd00)
    code = a.finish(); assert DRAW + len(code) <= FINISH
    patches.append((DRAW, code))

    a = Asm(FINISH)
    # Return value 2 means a swap: keep holding the displaced item. Value 1
    # is a completed native move/merge and clears the overlay as before.
    a.cmp(0, 2); a.cond(1, 'native'); a.h(0x4770)
    a.label('native'); a.ldr(3, 0x1675a1); a.h(0x4718)
    code = a.finish(); assert FINISH + len(code) <= 0x2af000
    patches.append((FINISH, code))
    patches += [(0x111bc8, bl(0x111bc8, MOVE)),
                (0x111bd2, bl(0x111bd2, FINISH)),
                (0x1679e4, bl(0x1679e4, DRAW))]
    return patches

if __name__ == '__main__':
    for address, code in build(): print(hex(address), len(code), code.hex())
