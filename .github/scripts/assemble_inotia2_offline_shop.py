# Newly assembled title-scoped Thumb helpers; no game assets or private fixtures.
import struct

class Asm:
 def __init__(self,base): self.base=base;self.b=bytearray();self.labels={};self.fix=[];self.lits=[]
 @property
 def pc(self):return self.base+len(self.b)
 def h(self,v):self.b+=struct.pack('<H',v)
 def label(self,s):self.labels[s]=self.pc
 def mov(self,r,v):self.h(0x2000|(r<<8)|v)
 def reg(self,d,s):self.h(0x4600|((d&8)<<4)|(s<<3)|(d&7))
 def ldr(self,r,val):self.lits.append((len(self.b),r,val));self.h(0)
 def load(self,d,n,off=0):self.h(0x6800|((off//4)<<6)|(n<<3)|d)
 def store(self,d,n,off=0):self.h(0x6000|((off//4)<<6)|(n<<3)|d)
 def byte(self,d,n,off=0,load=False):self.h((0x7800 if load else 0x7000)|(off<<6)|(n<<3)|d)
 def half(self,d,n,off=0,load=False):self.h((0x8800 if load else 0x8000)|((off//2)<<6)|(n<<3)|d)
 def cmp(self,r,v):self.h(0x2800|(r<<8)|v)
 def add(self,r,v):self.h(0x3000|(r<<8)|v)
 def cond(self,cond,label):self.fix.append((len(self.b),cond,label));self.h(0)
 def branch(self,label):self.fix.append((len(self.b),None,label));self.h(0)
 def bl(self,to):d=to-self.pc-4;assert d%2==0 and -(1<<22)<=d<(1<<22);self.h(0xf000|((d>>12)&0x7ff));self.h(0xf800|((d>>1)&0x7ff))
 def finish(self):
  if self.pc%4:self.h(0x46c0)
  pool={}
  for pos,r,val in self.lits:
   if val not in pool:pool[val]=self.pc;self.b+=struct.pack('<I',val)
   delta=pool[val]-((self.base+pos+4)&~3);assert delta>=0 and delta%4==0 and delta<=1020
   struct.pack_into('<H',self.b,pos,0x4800|(r<<8)|(delta//4))
  for pos,cond,label in self.fix:
   delta=self.labels[label]-self.base-pos-4;assert delta%2==0
   if cond is None:assert -2048<=delta<=2046;v=0xe000|((delta//2)&0x7ff)
   else:assert -256<=delta<=254;v=0xd000|(cond<<8)|((delta//2)&255)
   struct.pack_into('<H',self.b,pos,v)
  return bytes(self.b)

FLAG=0x2ad400
CATALOG=[(857,10000),(665,10000),(835,5000),(836,10000),(839,10000)]+[(i,20000) for i in range(943,948)]+[(949,20000)]
# Retain the original eleven in order, then append a curated premium-oriented
# selection. Equipment requirements and native pack contents remain unchanged.
CATALOG += [(941,50000),(940,50000),(950,30000),(951,30000),
            (647,5000),(837,10000),(838,10000),(4,30000),
            (933,50000),(934,100000),(935,30000),(936,30000),(880,10000)]
ENTRY=0x2ae200;INIT=0x2ae240;PRICE=0x2ae500;CLEANUP=0x2ae580;TABLE=0x2ae680;TITLE=0x2ae780

def jump(to):return bytes.fromhex('004b1847')+struct.pack('<I',to|1)
def bl(at,to):a=Asm(at);a.bl(to);return bytes(a.b)

def build():
 assert len(CATALOG) == 24 and len({i for i,_ in CATALOG}) == len(CATALOG)
 assert TABLE + len(CATALOG)*8 <= TITLE < 0x2af000
 patches=[]
 a=Asm(ENTRY);a.h(0xb510);a.ldr(4,FLAG);a.mov(0,1);a.store(0,4);a.mov(0,16);a.bl(0x176fd8);a.h(0xbd10)
 patches.append((ENTRY,a.finish()))
 a=Asm(INIT)
 # Preserve the original prologue on both paths; r8/r10 are native callee saved.
 a.h(0xb570);a.reg(6,10);a.reg(5,8);a.h(0xb460)
 a.ldr(3,FLAG);a.load(3,3);a.cmp(3,0);a.cond(1,'local')
 a.ldr(3,0x1205b1);a.h(0x4718)
 a.label('local');a.bl(0x167e10);a.bl(0x16816c);a.reg(4,0);a.cmp(4,0);a.cond(0,'done')
 a.bl(0x13f8a4);a.ldr(5,TABLE);a.mov(6,0)
 a.label('items');a.h(0x8828);a.bl(0x14a0bc);a.cmp(0,0);a.cond(0,'next');a.reg(1,6);a.bl(0x13f6b0)
 a.label('next');a.add(5,8);a.add(6,1);a.cmp(6,len(CATALOG));a.cond(3,'items')
 a.ldr(5,0x1924c4)
 a.ldr(3,0x1924c4+0x1048);a.load(3,3);a.byte(0,3,load=True);a.ldr(2,FLAG+12);a.store(0,2);a.mov(0,1);a.byte(0,3)
 a.mov(0,0);a.store(0,4,16)
 for off,val in [(1,6),(12,len(CATALOG)),(7,20),(8,20),(4,4),(2,4),(3,6),(11,0)]:a.mov(0,val);a.byte(0,4,off)
 for off,slot in [(28,0x5b4),(32,0x5b8),(36,0x1064)]:a.ldr(3,0x1924c4+slot);a.load(3,3);a.store(3,4,off)
 a.ldr(3,0x1924c4+0x598);a.load(3,3);a.store(4,3)
 a.mov(0,0);a.bl(0x11fc90);a.reg(0,4);a.mov(1,0);a.bl(0x167ef0);a.reg(1,0)
 a.ldr(3,0x2aaa78);a.load(2,3);a.mov(0,0);a.bl(0x164b24)
 a.mov(0,0);a.ldr(1,TITLE);a.bl(0x168ec8)
 a.label('done');a.h(0xbc0c);a.reg(8,2);a.reg(10,3);a.h(0xbd70)
 patches.append((INIT,a.finish()))
 a=Asm(PRICE);a.h(0xb511);a.ldr(3,FLAG);a.load(3,3);a.cmp(3,0);a.cond(0,'normal')
 a.h(0x8902);a.h(0x0992);a.ldr(3,TABLE);a.mov(1,len(CATALOG))
 a.label('price');a.h(0x8818);a.h(0x4290);a.cond(0,'found');a.add(3,8);a.h(0x3901);a.cond(1,'price')
 a.branch('normal');a.label('found');a.load(0,3,4);a.h(0xb001);a.h(0xbd10)
 a.label('normal');a.h(0xbc11);a.h(0xbc08);a.reg(14,3);a.h(0xb570);a.reg(5,0);a.bl(0x149f00);a.ldr(3,0x14aed1);a.h(0x4718)
 patches.append((PRICE,a.finish()))
 a=Asm(CLEANUP);a.h(0xb510);a.ldr(4,FLAG);a.load(0,4);a.cmp(0,0);a.cond(0,'clean');a.ldr(3,0x1924c4+0x1048);a.load(3,3);a.load(0,4,12);a.byte(0,3);a.mov(0,0);a.store(0,4)
 a.label('clean');a.bl(0x13f8a4);a.h(0xbd10)
 patches.append((CLEANUP,a.finish()))
 patches +=[(TABLE,b''.join(struct.pack('<II',i,p) for i,p in CATALOG)),(TITLE,'로컬 골드 상점'.encode('cp949')+b'\0')]
 patches +=[(0x120890,jump(ENTRY)),(0x1205a8,jump(INIT)),(0x14aec8,jump(PRICE)),(0x11fec8,bl(0x11fec8,CLEANUP))]
 return patches

if __name__ == '__main__':
 for address, data in build():
  print(f'{address:#08x}: {data.hex()}')
