# Authored Thumb hook for the known Inotia2 client; no extracted game code.
from assemble_inotia2_offline_shop import Asm, jump

HELPER = 0x2ae800
HOOK = 0x1476cc

def build():
    a = Asm(HELPER)
    # The native acquisition routine has already accepted bag capacity. r6
    # still owns the live incoming object; stack insertion has not freed it.
    # Invoke the same predicate/action used by item-identification scrolls.
    a.h(0xb502)                 # push {r1,lr}; preserve native caller linkage
    a.reg(0, 6)
    a.bl(0x14a064)              # native identify: non-equipment is unchanged
    # Identification may resolve the native starter dagger's item ID. Refresh
    # the cached notification ID as well as the original quantity contract.
    a.half(5, 6, 8, load=True)
    a.h(0x09ad)                 # lsr r5,r5,#6
    a.reg(0, 6)
    a.bl(0x147eb4)              # native item quantity, one for equipment
    a.h(0xbc0a)                 # pop {r1,r3}
    a.reg(14, 3)
    a.byte(2, 7, load=True)     # displaced native slot load
    a.ldr(3, 0x1476d5)
    a.h(0x4718)                # resume original insertion/notification
    code = a.finish()
    assert HELPER + len(code) <= 0x2af000
    return [(HELPER, code), (HOOK, jump(HELPER))]

if __name__ == '__main__':
    for address, code in build():
        print(f'{address:#08x}: {code.hex()}')
