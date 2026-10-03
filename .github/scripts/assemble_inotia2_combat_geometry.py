# Authored rectangle intersection; no extracted native game code.
from assemble_inotia2_offline_shop import Asm, jump

HELPER = 0x2ae900
HOOK = 0x1734bc

def build():
    a = Asm(HELPER)
    a.h(0xb5f0)
    def signed_half(rd, base, offset):
        a.h(0x5e00 | (offset << 6) | (base << 3) | rd)
    def compare(left, right):
        a.h(0x4280 | (right << 3) | left)
    a.mov(2, 0)
    signed_half(6, 0, 2); signed_half(7, 1, 2)
    a.mov(2, 4)
    signed_half(3, 0, 2); signed_half(4, 1, 2)
    compare(3, 7); a.cond(11, 'zero')
    compare(6, 4); a.cond(12, 'zero')
    compare(6, 7); a.cond(10, 'left'); a.reg(6, 7)
    a.label('left')
    compare(3, 4); a.cond(13, 'right'); a.reg(3, 4)
    a.label('right'); a.h(0x1b9e)
    a.mov(2, 2)
    signed_half(7, 0, 2); signed_half(5, 1, 2)
    a.mov(2, 6)
    signed_half(3, 0, 2); signed_half(4, 1, 2)
    compare(3, 5); a.cond(11, 'zero')
    compare(7, 4); a.cond(12, 'zero')
    compare(7, 5); a.cond(10, 'top'); a.reg(7, 5)
    a.label('top')
    compare(3, 4); a.cond(13, 'bottom'); a.reg(3, 4)
    a.label('bottom'); a.h(0x1bdf); a.h(0x437e)
    a.reg(0, 6); a.h(0xbdf0)
    a.label('zero'); a.mov(0, 0); a.h(0xbdf0)
    code = a.finish()
    assert HELPER + len(code) < 0x2af000
    return [(HELPER, code), (HOOK, jump(HELPER))]

if __name__ == '__main__':
    for address, code in build():
        print(f'{address:#08x}: {code.hex()}')
