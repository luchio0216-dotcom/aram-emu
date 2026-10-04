# Authored Thumb call wrapper; no game code, assets, or private save fixtures.
from assemble_inotia2_offline_shop import Asm, bl

HELPER = 0x2aea00
HOOK = 0x154cc8

def build():
    a = Asm(HELPER)
    # This call belongs only to monster rewards. r10 still holds the defeated
    # monster's native level, while r1 is the positive reward after the game's
    # party-size and recipient-level adjustments. Quest rewards use another call.
    a.reg(2, 10)
    a.cmp(2, 41)
    a.cond(11, 'native')
    a.h(0x0089)  # lsl r1,r1,#2
    a.label('native')
    a.ldr(3, 0x135551)
    a.h(0x4718)  # tail call; preserve the caller's original return address
    code = a.finish()
    assert HELPER + len(code) <= 0x2af000
    return [(HELPER, code), (HOOK, bl(HOOK, HELPER))]

if __name__ == '__main__':
    for address, code in build():
        print(f'{address:#08x}: {code.hex()}')
