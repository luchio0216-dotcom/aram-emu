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
ENTRY=0x2ae200;INIT=0x2ae240;PRICE=0x2ae500;CLEANUP=0x2ae580;TITLE=0x2ae780
TABLE=0x2ad500
BUILD_STOCK=0x2ad8e0;CLEAR_STOCK=0x2ada60;NAVIGATE=0x2adb00;SCROLL=0x2adb80;RESOLVE=0x2adc20
CATEGORIES=0x2add10;MENU_INIT=0x2add50;MENU_INPUT=0x2ade20;MENU_NAMES=0x2adf10;MENU_TEXT=0x2adf80;MENU_ITEMS=0x2adf60
SELECT_R5=0x2adc80;SELECT_R0=0x2adcb0;SELECT_R6=0x2adce0
# Numeric equipment references, ordered by category, subtype, required level,
# and ID. Definitions, names, icons, options and restrictions come from the
# user's guarded game image. No extracted game files are embedded.
EQUIPMENT=(
 (504, 0, 27, 0),
 (922, 0, 30, 0),
 (721, 0, 50, 0),
 (722, 0, 100, 0),
 (723, 0, 100, 0),
 (724, 0, 100, 0),
 (954, 0, 110, 0),
 (725, 0, 25, 1),
 (938, 0, 27, 1),
 (579, 0, 33, 1),
 (580, 0, 33, 1),
 (668, 0, 40, 1),
 (726, 0, 70, 1),
 (727, 0, 90, 1),
 (928, 0, 98, 1),
 (728, 0, 100, 1),
 (729, 0, 100, 1),
 (941, 0, 100, 1),
 (953, 0, 110, 1),
 (730, 0, 75, 2),
 (731, 0, 100, 2),
 (732, 0, 100, 2),
 (733, 0, 45, 3),
 (734, 0, 100, 3),
 (735, 0, 100, 3),
 (736, 0, 100, 3),
 (737, 0, 55, 4),
 (927, 0, 95, 4),
 (738, 0, 100, 4),
 (739, 0, 100, 4),
 (740, 0, 100, 4),
 (741, 0, 60, 5),
 (742, 0, 80, 5),
 (926, 0, 70, 6),
 (743, 0, 100, 6),
 (744, 0, 100, 6),
 (745, 0, 35, 7),
 (925, 0, 56, 7),
 (746, 0, 100, 7),
 (747, 0, 100, 7),
 (748, 0, 100, 7),
 (952, 0, 110, 7),
 (503, 0, 27, 8),
 (749, 0, 65, 8),
 (750, 0, 100, 8),
 (751, 0, 100, 8),
 (923, 0, 40, 9),
 (752, 0, 85, 9),
 (753, 0, 100, 9),
 (754, 0, 100, 9),
 (956, 0, 110, 9),
 (505, 0, 27, 10),
 (755, 0, 40, 10),
 (939, 0, 40, 10),
 (929, 0, 98, 10),
 (756, 0, 100, 10),
 (957, 0, 110, 10),
 (937, 0, 27, 11),
 (940, 0, 40, 11),
 (924, 0, 50, 11),
 (757, 0, 100, 11),
 (758, 0, 100, 11),
 (759, 0, 100, 11),
 (955, 0, 110, 11),
 (760, 0, 30, 12),
 (930, 0, 98, 12),
 (761, 0, 100, 12),
 (762, 0, 100, 12),
 (771, 1, 25, 13),
 (881, 1, 30, 13),
 (882, 1, 42, 13),
 (883, 1, 54, 13),
 (772, 1, 64, 13),
 (884, 1, 70, 13),
 (885, 1, 95, 13),
 (886, 1, 98, 13),
 (773, 1, 100, 13),
 (774, 1, 100, 13),
 (775, 1, 100, 13),
 (829, 1, 100, 13),
 (958, 1, 110, 13),
 (776, 1, 30, 14),
 (521, 1, 33, 14),
 (887, 1, 35, 14),
 (888, 1, 45, 14),
 (889, 1, 56, 14),
 (777, 1, 68, 14),
 (890, 1, 80, 14),
 (891, 1, 98, 14),
 (778, 1, 100, 14),
 (779, 1, 100, 14),
 (780, 1, 100, 14),
 (959, 1, 110, 14),
 (781, 1, 35, 15),
 (892, 1, 38, 15),
 (893, 1, 50, 15),
 (894, 1, 58, 15),
 (782, 1, 72, 15),
 (895, 1, 85, 15),
 (896, 1, 95, 15),
 (897, 1, 98, 15),
 (783, 1, 100, 15),
 (784, 1, 100, 15),
 (785, 1, 100, 15),
 (960, 1, 110, 15),
 (786, 1, 40, 16),
 (898, 1, 40, 16),
 (899, 1, 53, 16),
 (900, 1, 65, 16),
 (787, 1, 76, 16),
 (901, 1, 90, 16),
 (902, 1, 98, 16),
 (788, 1, 100, 16),
 (789, 1, 100, 16),
 (790, 1, 100, 16),
 (961, 1, 110, 16),
 (948, 1, 1, 17),
 (950, 1, 1, 17),
 (703, 1, 28, 17),
 (691, 1, 32, 17),
 (704, 1, 38, 17),
 (684, 1, 40, 17),
 (692, 1, 42, 17),
 (705, 1, 48, 17),
 (685, 1, 50, 17),
 (693, 1, 52, 17),
 (706, 1, 58, 17),
 (675, 1, 60, 17),
 (686, 1, 60, 17),
 (694, 1, 62, 17),
 (676, 1, 65, 17),
 (707, 1, 68, 17),
 (677, 1, 70, 17),
 (687, 1, 70, 17),
 (695, 1, 72, 17),
 (678, 1, 75, 17),
 (708, 1, 78, 17),
 (679, 1, 80, 17),
 (696, 1, 82, 17),
 (680, 1, 85, 17),
 (709, 1, 88, 17),
 (689, 1, 90, 17),
 (682, 1, 95, 17),
 (710, 1, 98, 17),
 (711, 1, 100, 17),
 (712, 1, 100, 17),
 (951, 1, 1, 18),
 (713, 1, 33, 18),
 (670, 1, 35, 18),
 (699, 1, 37, 18),
 (671, 1, 40, 18),
 (672, 1, 45, 18),
 (673, 1, 50, 18),
 (714, 1, 53, 18),
 (674, 1, 55, 18),
 (700, 1, 57, 18),
 (715, 1, 73, 18),
 (701, 1, 77, 18),
 (688, 1, 80, 18),
 (681, 1, 90, 18),
 (697, 1, 92, 18),
 (716, 1, 93, 18),
 (683, 1, 100, 18),
 (690, 1, 100, 18),
 (698, 1, 100, 18),
 (702, 1, 100, 18),
 (717, 1, 100, 18),
 (718, 1, 100, 18),
 (903, 1, 30, 19),
 (791, 1, 44, 19),
 (904, 1, 53, 19),
 (905, 1, 70, 19),
 (792, 1, 80, 19),
 (906, 1, 98, 19),
 (793, 1, 100, 19),
 (794, 1, 100, 19),
 (795, 1, 100, 19),
 (963, 1, 105, 19),
 (907, 1, 38, 20),
 (796, 1, 48, 20),
 (908, 1, 54, 20),
 (909, 1, 80, 20),
 (797, 1, 84, 20),
 (910, 1, 98, 20),
 (798, 1, 100, 20),
 (799, 1, 100, 20),
 (800, 1, 100, 20),
 (962, 1, 105, 20),
 (763, 1, 20, 21),
 (919, 1, 50, 21),
 (920, 1, 98, 21),
 (764, 1, 100, 21),
 (765, 1, 100, 21),
 (766, 1, 100, 21),
 (966, 1, 105, 21),
 (767, 1, 60, 22),
 (921, 1, 65, 22),
 (768, 1, 100, 22),
 (769, 1, 100, 22),
 (770, 1, 100, 22),
 (590, 2, 40, 23),
 (911, 2, 42, 23),
 (801, 2, 52, 23),
 (912, 2, 56, 23),
 (913, 2, 85, 23),
 (802, 2, 88, 23),
 (914, 2, 98, 23),
 (803, 2, 100, 23),
 (804, 2, 100, 23),
 (805, 2, 100, 23),
 (964, 2, 105, 23),
 (915, 2, 45, 24),
 (806, 2, 56, 24),
 (916, 2, 58, 24),
 (917, 2, 90, 24),
 (807, 2, 92, 24),
 (918, 2, 98, 24),
 (808, 2, 100, 24),
 (809, 2, 100, 24),
 (810, 2, 100, 24),
 (965, 2, 105, 24),
)
CONSUMABLES=[(857,10000),(665,10000),(835,5000),(836,10000),(839,10000)]+[(i,20000) for i in range(943,948)]+[(949,20000)]
CONSUMABLES += [(647,5000),(837,10000),(838,10000),(4,30000),(933,50000),(934,100000),(935,30000),(936,30000),(880,10000)]
CONSUMABLES += [(812,10000)]
CONSUMABLES += [(i,10000) for i in (972,974,970,971,973,975)]
CATALOG=[(i,10000) for i,_,_,_ in EQUIPMENT]+CONSUMABLES
ROWS=(len(CATALOG)+5)//6

def jump(to):return bytes.fromhex('004b1847')+struct.pack('<I',to|1)
def bl(at,to):a=Asm(at);a.bl(to);return bytes(a.b)
def cmpreg(a,n,m):a.h(0x4280|(m<<3)|n)
def subreg(a,d,n,m):a.h(0x1a00|(m<<6)|(n<<3)|d)
def padded(data,length):assert len(data)<=length;return data+bytes(length-len(data))

def build():
 assert len(CATALOG)==248 and len({i for i,_ in CATALOG})==248
 assert TABLE+len(CATALOG)*4 <= BUILD_STOCK
 assert EQUIPMENT==tuple(sorted(EQUIPMENT,key=lambda x:(x[1],x[3],x[2],x[0])))
 patches=[]
 a=Asm(ENTRY);a.h(0xb510);a.ldr(4,FLAG);a.mov(0,1);a.store(0,4);a.mov(0,1);a.store(0,4,8);a.mov(0,16);a.bl(0x176fd8);a.h(0xbd10)
 patches.append((ENTRY,a.finish()))
 # Preserve the displaced merchant prologue and its normal branch. An
 # expanded stock list is allocated once, cleared on exit and reused.
 a=Asm(BUILD_STOCK);a.h(0xb570);a.reg(6,10);a.reg(5,8);a.h(0xb460)
 a.ldr(3,FLAG);a.load(3,3);a.cmp(3,0);a.cond(1,'local');a.ldr(3,0x1205b1);a.h(0x4718)
 a.label('local');a.ldr(3,FLAG+8);a.load(3,3);a.cmp(3,0);a.cond(0,'products');a.bl(MENU_INIT);a.branch('done');a.label('products');a.h(0xb480);a.bl(0x167e10);a.bl(0x16816c);a.reg(4,0);a.cmp(4,0);a.cond(0,'failed')
 a.ldr(3,FLAG+20);a.load(7,3);a.cmp(7,0);a.cond(1,'allocated')
 a.ldr(0,len(CATALOG)*8);a.bl(0x125c54);a.reg(7,0);a.cmp(7,0);a.cond(0,'release');a.ldr(3,FLAG+20);a.store(7,3)
 a.label('allocated');a.bl(0x13f8a4);a.ldr(3,0x1934f4);a.load(0,3);a.ldr(2,FLAG+16);a.store(0,2);a.store(7,3)
 a.ldr(3,FLAG+4);a.load(0,3);a.cmp(0,3);a.cond(9,'category');a.mov(0,0);a.label('category');a.h(0x0043);a.h(0x181b);a.h(0x009b);a.ldr(2,CATEGORIES);a.h(0x189b);a.load(5,3);a.load(0,3,4);a.ldr(2,FLAG+32);a.store(0,2);a.load(0,3,8);a.store(0,2,4);a.mov(6,0)
 a.label('items');a.half(0,5,load=True);a.bl(0x14a0bc);a.store(0,7,4);a.cmp(0,0);a.cond(0,'missing');a.mov(0,0);a.branch('slot');a.label('missing');a.mov(0,1)
 a.label('slot');a.store(0,7);a.add(7,8);a.add(5,4);a.add(6,1);a.ldr(3,FLAG+32);a.load(0,3);cmpreg(a,6,0);a.cond(3,'items')
 a.ldr(3,0x19350c);a.load(3,3);a.byte(0,3,load=True);a.ldr(2,FLAG+12);a.store(0,2);a.mov(0,1);a.byte(0,3)
 a.mov(0,0);a.store(0,4,16)
 for off,val in [(1,6),(7,20),(8,20),(4,4),(3,6),(11,0)]:a.mov(0,val);a.byte(0,4,off)
 a.byte(6,4,12);a.ldr(3,FLAG+36);a.load(0,3);a.byte(0,4,2)
 for off,slot in [(28,0x5b4),(32,0x5b8),(36,0x1064)]:a.ldr(3,0x1924c4+slot);a.load(3,3);a.store(3,4,off)
 a.ldr(3,0x192a5c);a.load(3,3);a.store(4,3);a.ldr(3,FLAG+24);a.store(4,3)
 a.mov(0,0);a.bl(0x11fc90);a.reg(0,4);a.mov(1,0);a.bl(0x167ef0);a.reg(1,0)
 a.ldr(3,0x2aaa78);a.load(2,3);a.mov(0,0);a.bl(0x164b24)
 a.mov(0,0);a.ldr(1,TITLE);a.bl(0x168ec8);a.branch('localdone')
 a.label('release');a.reg(0,4);a.bl(0x1681a8)
 a.label('failed');a.ldr(3,FLAG);a.mov(0,0);a.store(0,3);a.mov(0,3);a.bl(0x176fd8)
 a.label('localdone');a.h(0xbc80);a.label('done');a.h(0xbc0c);a.reg(8,2);a.reg(10,3);a.h(0xbd70)
 code=a.finish();assert BUILD_STOCK+len(code)<=CLEAR_STOCK;patches.append((BUILD_STOCK,code))
 patches.append((INIT,padded(jump(BUILD_STOCK),256)))
 a=Asm(PRICE);a.h(0xb511);a.ldr(3,FLAG);a.load(3,3);a.cmp(3,0);a.cond(0,'normal')
 a.h(0x8902);a.h(0x0992);a.ldr(3,TABLE);a.mov(1,len(CATALOG))
 a.label('price');a.half(0,3,load=True);cmpreg(a,0,2);a.cond(0,'found');a.add(3,4);a.h(0x3901);a.cond(1,'price')
 a.branch('normal');a.label('found');a.half(0,3,2,load=True);a.ldr(1,1000);a.h(0x4348);a.h(0xb001);a.h(0xbd10)
 a.label('normal');a.h(0xbc11);a.h(0xbc08);a.reg(14,3);a.h(0xb570);a.reg(5,0);a.bl(0x149f00);a.ldr(3,0x14aed1);a.h(0x4718)
 patches.append((PRICE,a.finish()))
 # Match native cleanup's ownership rule: never free a retained buyback item.
 a=Asm(CLEAR_STOCK);a.h(0xb570);a.ldr(4,FLAG);a.load(0,4);a.cmp(0,0);a.cond(0,'normal')
 a.load(0,4,24);a.cmp(0,0);a.cond(0,'presentation');a.load(5,4,20);a.cmp(5,0);a.cond(0,'restore');a.load(6,4,32)
 a.label('clear');a.load(0,5,4);a.cmp(0,0);a.cond(0,'empty');a.bl(0x13f7a8);a.cmp(0,0);a.cond(1,'empty');a.load(0,5,4);a.bl(0x147988)
 a.label('empty');a.mov(0,1);a.store(0,5);a.mov(0,0);a.store(0,5,4);a.add(5,8);a.h(0x3e01);a.cond(1,'clear')
 a.label('restore');a.load(0,4,16);a.cmp(0,0);a.cond(0,'presentation');a.ldr(3,0x1934f4);a.store(0,3)
 a.label('presentation');a.ldr(3,0x19350c);a.load(3,3);a.load(0,4,12);a.byte(0,3);a.mov(0,0);a.store(0,4);a.store(0,4,24)
 a.label('normal');a.bl(0x13f8a4);a.h(0xbd70)
 code=a.finish();assert CLEAR_STOCK+len(code)<=NAVIGATE;patches.append((CLEAR_STOCK,code));patches.append((CLEANUP,padded(jump(CLEAR_STOCK),40)))
 # The game's navigation arithmetic accepts full integers; its stock-widget
 # wrapper narrows them to int8. Keep uint8 indices only for our exact widget.
 a=Asm(NAVIGATE);a.ldr(3,FLAG+24);a.load(3,3);cmpreg(a,0,3);a.cond(0,'local')
 a.h(0xb5f0);a.reg(6,0);a.mov(3,11);a.h(0x56f3);a.ldr(2,0x168285);a.h(0x4710)
 a.label('local');a.h(0xb550);a.h(0xb083);a.reg(4,0);a.reg(6,1);a.byte(3,4,11,load=True);a.h(0x9301);a.h(0xab02);a.h(0x9300)
 a.reg(0,6);a.byte(1,4,1,load=True);a.byte(2,4,2,load=True);a.h(0xab01);a.bl(0x14b35c);a.cmp(0,0);a.cond(0,'done')
 a.h(0x9b01);a.byte(3,4,11);a.reg(0,4);a.bl(SCROLL);a.mov(0,1)
 a.label('done');a.h(0xb003);a.h(0xbd50)
 code=a.finish();assert NAVIGATE+len(code)<=SCROLL;patches.append((NAVIGATE,code));patches.append((0x16827c,jump(NAVIGATE)))
 a=Asm(SCROLL);a.h(0xb570);a.reg(4,0);a.byte(0,4,11,load=True);a.byte(1,4,1,load=True);a.bl(0x183db4);a.reg(5,0)
 a.byte(3,4,6,load=True);cmpreg(a,5,3);a.cond(3,'up');a.byte(2,4,4,load=True);a.h(0x189b);a.h(0x3b01);cmpreg(a,5,3);a.cond(9,'done')
 subreg(a,5,5,2);a.add(5,1)
 a.label('up');a.byte(5,4,6)
 a.label('done');a.h(0xbd70)
 code=a.finish();assert SCROLL+len(code)<=RESOLVE;patches.append((SCROLL,code))
 a=Asm(RESOLVE);a.ldr(3,FLAG+24);a.load(3,3);cmpreg(a,0,3);a.cond(1,'original');a.h(0x0609);a.h(0x0e09)
 # The displaced prologue includes the original null-return instruction at
 # 0x167ef6. Native positive bounds failures branch back there too, so check
 # BOTH bounds before returning to the unmodified resolver body. Otherwise
 # an out-of-range cell executes jump-literal bytes and returns the widget
 # pointer as an item in every merchant, causing bogus icons and a freeze.
 a.label('original');a.h(0xb500);a.cmp(1,0);a.cond(11,'empty')
 a.byte(3,0,12,load=True);cmpreg(a,1,3);a.cond(2,'empty')
 a.ldr(3,0x167efb);a.h(0x4718)
 a.label('empty');a.mov(0,0);a.h(0xbd00)
 code=a.finish();assert RESOLVE+len(code)<=SELECT_R5;patches.append((RESOLVE,code));patches.append((0x167ef0,jump(RESOLVE)))
 for address,reg,sites,end in [(SELECT_R5,5,(0x167f98,0x167fcc),SELECT_R0),(SELECT_R0,0,(0x120690,),SELECT_R6),(SELECT_R6,6,(0x120404,0x168280,0x1682a0),0x2add10)]:
  a=Asm(address);a.h(0xb403);a.ldr(1,FLAG+24);a.load(1,1);cmpreg(a,reg,1);a.byte(3,reg,11,load=True);a.cond(0,'done');a.h(0x061b);a.h(0x161b)
  a.label('done');a.h(0xbc03);a.h(0x4770);code=a.finish();assert address+len(code)<=end;patches.append((address,code))
  # 0x168280 lies inside the displaced navigation prologue and is already
  # represented by its whole-function guard; no overlapping patch is emitted.
  for site in sites:
   if site!=0x168280:patches.append((site,bl(site,address)))
 # Four categories reuse the native merchant frame and list painter. The
 # menu owns no stock items; OK initializes a category in the same state,
 # and Clear returns to the category menu through native widget cleanup.
 groups=[(221,27),(0,68),(68,132),(200,21)]
 patches.append((CATEGORIES,b''.join(struct.pack('<III',TABLE+offset*4,count,(count+5)//6) for offset,count in groups)))
 a=Asm(MENU_INIT);a.h(0xb510);a.bl(0x167e10);a.bl(0x16816c);a.reg(4,0);a.cmp(4,0);a.cond(0,'done')
 a.bl(0x13f8a4);a.ldr(3,0x19350c);a.load(3,3);a.byte(0,3,load=True);a.ldr(2,FLAG+12);a.store(0,2);a.mov(0,1);a.byte(0,3)
 for off,val in [(1,1),(2,4),(3,1),(4,4),(7,132),(8,20),(11,0),(12,4)]:a.mov(0,val);a.byte(0,4,off)
 a.ldr(0,MENU_ITEMS);a.store(0,4,16);a.ldr(0,0x12066d);a.store(0,4,28)
 a.ldr(3,0x192a5c);a.load(3,3);a.store(4,3);a.ldr(3,FLAG+24);a.mov(0,0);a.store(0,3);a.bl(0x11fc90)
 a.mov(0,0);a.mov(1,0);a.ldr(3,0x2aaa78);a.load(2,3);a.bl(0x164b24);a.mov(0,0);a.ldr(1,TITLE);a.bl(0x168ec8)
 a.label('done');a.h(0xbd10);code=a.finish();assert MENU_INIT+len(code)<=MENU_INPUT;patches.append((MENU_INIT,code))
 a=Asm(MENU_INPUT);a.h(0xb570);a.reg(4,0);a.ldr(5,FLAG);a.load(0,5);a.cmp(0,0);a.cond(0,'original')
 a.ldr(3,0x1928e8);a.load(6,3);a.load(0,6,72);cmpreg(a,4,0);a.cond(0,'back')
 a.load(0,5,8);a.cmp(0,0);a.cond(0,'original')
 a.load(0,6,16);cmpreg(a,4,0);a.cond(0,'choose')
 a.ldr(3,0x192a5c);a.load(3,3);a.load(0,3);a.reg(1,4);a.bl(0x16827c);a.branch('done')
 a.label('choose');a.ldr(3,0x192a5c);a.load(3,3);a.load(3,3);a.byte(0,3,11,load=True);a.cmp(0,3);a.cond(8,'done');a.store(0,5,4)
 a.bl(0x11fec0);a.mov(0,1);a.store(0,5);a.mov(0,0);a.store(0,5,8);a.bl(BUILD_STOCK);a.branch('done')
 a.label('back');a.load(6,5,8);a.bl(0x11fec0);a.cmp(6,0);a.cond(1,'leave')
 a.mov(0,1);a.store(0,5);a.store(0,5,8);a.bl(MENU_INIT);a.branch('done')
 a.label('leave');a.mov(0,0);a.store(0,5,8);a.mov(0,2);a.bl(0x176fd8);a.branch('done')
 a.label('original');a.reg(0,4);a.h(0xbc70);a.h(0xbc08);a.reg(14,3)
 a.h(0xb5f0);a.reg(7,10);a.reg(6,8);a.h(0xb4c0);a.ldr(3,0x1203a1);a.h(0x4718)
 a.label('done');a.h(0xbd70);code=a.finish();assert MENU_INPUT+len(code)<=MENU_NAMES;patches.append((MENU_INPUT,code));patches.append((0x120398,jump(MENU_INPUT)))
 # The existing title-scoped text dispatcher tail-calls this helper after
 # handling class skillbook strings. Other text keeps its original lookup.
 a=Asm(MENU_NAMES);a.ldr(2,FLAG+8);a.load(2,2);a.cmp(2,1);a.cond(1,'original');a.ldr(3,4961);subreg(a,2,0,3);a.cmp(2,3);a.cond(8,'original')
 a.h(0x0092);a.ldr(3,MENU_TEXT);a.h(0x5898);a.h(0x4770)
 a.label('original');a.h(0xb500);a.ldr(2,0x1924c4);a.ldr(3,0x1760);a.ldr(1,0x14fd15);a.h(0x4708)
 code=a.finish();assert MENU_NAMES+len(code)<=MENU_ITEMS;patches.append((MENU_NAMES,code))
 patches.append((MENU_ITEMS,struct.pack('<4I',28,29,30,31)))
 names=['소비템','무기','장비','장신구'];data=bytearray();ptrs=[]
 for name in names:ptrs.append(MENU_TEXT+16+len(data));data+=name.encode('cp949')+bytes([0])
 data=struct.pack('<4I',*ptrs)+data;assert MENU_TEXT+len(data)<=0x2ae000;patches.append((MENU_TEXT,data))
 patches += [(TABLE,b''.join(struct.pack('<HH',i,p//1000) for i,p in CATALOG)),(TITLE,'로컬 골드 상점'.encode('cp949')+b'\0')]
 patches += [(0x120890,jump(ENTRY)),(0x1205a8,jump(INIT)),(0x14aec8,jump(PRICE)),(0x11fec8,bl(0x11fec8,CLEANUP))]
 return patches

if __name__=='__main__':
 for address,data in build():print(f'{address:#08x}: {data.hex()}')
