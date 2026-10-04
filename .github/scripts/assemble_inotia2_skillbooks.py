# Authored Thumb wrappers. Item definitions are cloned at runtime from the
# user's exact client; no original game data or media is embedded here.
import struct
from assemble_inotia2_offline_shop import Asm, jump, bl

ENSURE=0x2aea40
BOOT=0x2aeb00
CREATE=0x2aeb20
USE=0x2aeb60
NAME=0x2aec10
STRING=0x2aec50
TEXT_TABLE=0x2aec80
TEXT=0x2aed00
FIRST=970
COUNT=976
CLASSES=['바바리안','템플러','로그','쉐도우헌터','프리스트','아크메이지']

def sub(a,d,n,m):a.h(0x1a00|(m<<6)|(n<<3)|d)
def cmp(a,n,m):a.h(0x4280|(m<<3)|n)

def build():
 patches=[]
 a=Asm(ENSURE)
 a.h(0xb5f0);a.ldr(4,0x194824);a.half(0,4,load=True);a.ldr(3,FIRST);cmp(a,0,3);a.cond(1,'done')
 a.ldr(5,0x194828);a.load(6,5);a.cmp(6,0);a.cond(0,'done')
 a.ldr(0,COUNT*17);a.bl(0x125c54);a.cmp(0,0);a.cond(0,'done');a.reg(7,0)
 a.reg(1,6);a.ldr(2,FIRST*17);a.bl(0x183b64)
 a.ldr(3,839*17);a.h(0x18f6) # add r6,r6,r3: native sealed-book definition
 a.ldr(3,FIRST*17);a.h(0x18fb) # add r3,r7,r3
 a.reg(4,3);a.mov(5,6)
 a.label('append');a.reg(0,4);a.reg(1,6);a.mov(2,17);a.bl(0x183b64);a.ldr(3,60006);sub(a,3,3,5);a.half(3,4);a.add(4,17);a.h(0x3d01);a.cond(1,'append')
 # Publish only the completed table. Allocation failure keeps original bounds.
 a.ldr(3,0x194828);a.store(7,3);a.ldr(3,0x194824);a.ldr(0,COUNT);a.half(0,3)
 a.label('done');a.h(0xbdf0)
 code=a.finish();assert ENSURE+len(code)<=BOOT;patches.append((ENSURE,code))
 a=Asm(BOOT);a.h(0xb510);a.bl(0x101950);a.h(0xb40f);a.bl(ENSURE);a.h(0xbc0f);a.h(0xbd10)
 code=a.finish();assert BOOT+len(code)<=CREATE;patches.append((BOOT,code))
 a=Asm(CREATE);a.h(0xb50f);a.bl(ENSURE);a.h(0xbc0f);a.h(0xbc08);a.reg(14,3)
 # Recreate the displaced constructor prologue and retain its argument.
 a.h(0xb5f0);a.reg(7,10);a.reg(6,8);a.h(0xb4c0);a.ldr(3,0x14a0c5);a.h(0x4718)
 code=a.finish();assert CREATE+len(code)<=USE;patches.append((CREATE,code))
 a=Asm(USE);a.h(0xb570);a.reg(4,0);a.ldr(3,FIRST);sub(a,4,4,3);a.cmp(4,5);a.cond(8,'original')
 a.ldr(0,366);a.bl(0x14a0bc);a.reg(5,0);a.cmp(5,0);a.cond(0,'failed')
 # Twelve native book-eligible skills per class: 0..5 and 8..13. The native
 # RNG remains responsible for the choice, with no reroll loop or frame hook.
 a.mov(0,0);a.mov(1,11);a.bl(0x125bb4);a.cmp(0,6);a.cond(3,'skill');a.add(0,2)
 a.label('skill');a.h(0x0123);a.h(0x181b) # class*16 + chosen native skill offset
 a.reg(0,5);a.mov(1,0);a.mov(2,29);a.bl(0x147b74);a.cmp(0,0);a.cond(0,'release')
 a.reg(0,5);a.bl(0x147688);a.cmp(0,0);a.cond(0,'release');a.mov(0,1);a.h(0xbd70)
 a.label('release');a.reg(0,5);a.bl(0x147988)
 a.label('failed');a.mov(0,8);a.mov(1,0);a.mov(2,0);a.mov(3,0);a.bl(0x16986c);a.mov(0,0);a.h(0xbd70)
 a.label('original');a.h(0xbc70);a.h(0xbc08);a.reg(14,3)
 a.h(0xb530);a.ldr(3,0xfffffcd3);a.ldr(5,0x47f90);a.h(0x18c0);a.ldr(3,0x14a531);a.h(0x4718)
 code=a.finish();assert USE+len(code)<=NAME;patches.append((USE,code))
 a=Asm(NAME);a.h(0xb504);a.ldr(3,FIRST);sub(a,2,0,3);a.cmp(2,5);a.cond(8,'original')
 a.cmp(1,0);a.cond(0,'index');a.add(2,6)
 a.label('index');a.h(0x0092);a.ldr(3,TEXT_TABLE);a.h(0x5898);a.h(0xbd04)
 a.label('original');a.h(0xbc04);a.h(0xbc08);a.reg(14,3)
 a.h(0xb570);a.ldr(4,0x1924c4);a.h(0x0609);a.ldr(3,0x148ca5);a.h(0x4718)
 code=a.finish();assert NAME+len(code)<=STRING;patches.append((NAME,code))
 a=Asm(STRING);a.ldr(3,60000);sub(a,2,0,3);a.cmp(2,5);a.cond(8,'original');a.h(0x0092);a.ldr(3,TEXT_TABLE);a.h(0x5898);a.h(0x4770)
 a.label('original');a.h(0xb500);a.ldr(2,0x1924c4);a.ldr(3,0x1760);a.ldr(1,0x14fd15);a.h(0x4708)
 code=a.finish();assert STRING+len(code)<=TEXT_TABLE;patches.append((STRING,code))
 texts=[f'봉인된 스킬북({c})' for c in CLASSES]+[f'사용하면 {c}의 스킬북 하나를 무작위로 얻습니다.' for c in CLASSES]
 data=bytearray();pointers=[]
 for t in texts:pointers.append(TEXT+len(data));data+=t.encode('cp949')+b'\0'
 assert TEXT+len(data)<=0x2af000
 patches.extend([(TEXT_TABLE,struct.pack('<12I',*pointers)),(TEXT,bytes(data))])
 patches.extend([(0x1450be,bl(0x1450be,BOOT)),(0x14a0bc,jump(CREATE)),(0x14a528,jump(USE)),(0x148c9c,jump(NAME)),(0x14fd0c,jump(STRING))])
 return patches

if __name__=='__main__':
 for a,b in build():print(hex(a),len(b),b.hex())
