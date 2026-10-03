# Authored Thumb fix for a missing current target in the threat list.
from assemble_inotia2_offline_shop import Asm, jump
import struct

HELPER = 0x2ae980
HOOK = 0x13180a

def build():
    a = Asm(HELPER)
    # r4 is the already validated best threat index. The current target
    # lookup in r0 can return -1 after its entry disappears. Treat that as
    # selecting the best target, before any sixteen-byte table arithmetic.
    a.cmp(0, 0)
    a.cond(10, 'valid')
    a.reg(0, 4)
    a.label('valid')
    a.h(0x4284)  # displaced cmp r4,r0
    a.cond(0, 'same')
    a.ldr(1, 0x14f8)  # displaced GOT slot literal
    a.h(0x586b)       # displaced ldr r3,[r5,r1]
    a.load(3, 3)      # displaced ldr r3,[r3]
    # r2 is dead on both continuations: native code assigns it before use.
    a.ldr(2, 0x131815)
    a.h(0x4710)
    a.label('same')
    a.ldr(2, 0x13188f)
    a.h(0x4710)
    code = a.finish()
    assert HELPER + len(code) < 0x2af000
    # This hook is halfword-aligned. Pad before its aligned literal.
    hook = bytes.fromhex('014b1847c046') + struct.pack('<I', HELPER | 1)
    return [(HELPER, code), (HOOK, hook)]

if __name__ == '__main__':
    for address, code in build():
        print(f'{address:#08x}: {code.hex()}')
