package application

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/mirusu400/aram-core/cpu"
)

const inotia2OfflineShopFlag = uint32(0x2ad400)
const inotia2OfflineShopEntry = uint32(0x2ae200)
const inotia2OfflineShopInit = uint32(0x2ae240)
const inotia2OfflineShopPrice = uint32(0x2ae500)
const inotia2OfflineShopCleanup = uint32(0x2ae580)
const inotia2OfflineShopTable = uint32(0x2ad500)
const inotia2OfflineShopBuildStock = uint32(0x2ad8e0)
const inotia2OfflineShopClearStock = uint32(0x2ada60)
const inotia2OfflineShopMenuInit = uint32(0x2add50)
const inotia2OfflineShopMenuInput = uint32(0x2ade20)
const inotia2OfflineShopNavigate = uint32(0x2adb00)
const inotia2OfflineShopResolve = uint32(0x2adc20)

type inotia2OfflineShopPatch struct {
	address     uint32
	originalSHA string
	legacySHA   string
	previousSHA string
	recentSHA   string
	replacement []byte
}

// New code and numeric references use existing mapped padding. The stock
// array is allocated by the native allocator and reused after native cleanup.
// The original image size, bootstrap and save identity remain unchanged.
// Category entry, native merchant widgets and local purchase prices are
// scoped to this shop. Other widget indices keep their signed sentinel. The native buying transaction retains its
// cloning, capacity check, successful insertion and gold-debit ordering.
// Other merchant initialization/pricing falls through to the original body.
var inotia2OfflineShopPatches = []inotia2OfflineShopPatch{
	{address: 0x2ae200, originalSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", legacySHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", previousSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", recentSHA: "205e38a6e03d69c1695dd5ce242afb17dcd6c6af28a143dc857b3f5e95c6a6e8", replacement: inotia2PatchBytes("10b5044c012020600120a0601020c8f6e3fe10bd00d42a00")},
	{address: 0x2ad8e0, originalSHA: "08c3e5bbea4fba57201a69d4bf70a0d255df921fdef4924c22da087f9338c2bc", legacySHA: "08c3e5bbea4fba57201a69d4bf70a0d255df921fdef4924c22da087f9338c2bc", previousSHA: "08c3e5bbea4fba57201a69d4bf70a0d255df921fdef4924c22da087f9338c2bc", recentSHA: "08c3e5bbea4fba57201a69d4bf70a0d255df921fdef4924c22da087f9338c2bc", replacement: inotia2PatchBytes("70b55646454660b4484b1b68002b01d1474b1847474b1b68002b02d000f028fa7fe080b4baf684fabaf630fc0446002c70d0414b1f68002f07d1404878f69af90746002f63d03c4b1f6091f6bbff3c4b18683c4a10601f603b4b1868032800d9002043001b189b00384a9b181d685868374a106098685060002628889cf6aefb7860002801d0002000e0012038600837043501362e4b18688642eed32d4b1b6818782d4a10600120187000202061062060701420e07114202072042020710620e0700020e0722673244b1868a070244b1b68e361234b1b682362234b1b686362224b1b681c60224b1c60002072f660f920460021baf68cfa01461e4b1a680020b7f6a0f800201c49bbf66efa08e02046baf6dafb054b002018600320c9f6ecfa80bc0cbc90469a4670bdc04600d42a00b105120008d42a0014d42a00c0070000f434190010d42a0004d42a0010dd2a0020d42a000c3519000cd42a0024d42a00782a19007c2a1900283519005c2a190018d42a0078aa2a0080e72a00")},
	{address: 0x2ae240, originalSHA: "5341e6b2646979a70e57653007a1f310169421ec9bdd9f1a5648f75ade005af1", legacySHA: "5341e6b2646979a70e57653007a1f310169421ec9bdd9f1a5648f75ade005af1", previousSHA: "5341e6b2646979a70e57653007a1f310169421ec9bdd9f1a5648f75ade005af1", recentSHA: "0e37d34768e09a466b1f145e3aaae57eeb45cb7b9e94e1cfed9d88181f8bdb5f", replacement: inotia2PatchBytes("004b1847e1d82a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")},
	{address: 0x2ae500, originalSHA: "f2c0d5456a983ecd12e314fcfa19879179fc8424343baeb1325457472ae85601", legacySHA: "f2c0d5456a983ecd12e314fcfa19879179fc8424343baeb1325457472ae85601", previousSHA: "f2c0d5456a983ecd12e314fcfa19879179fc8424343baeb1325457472ae85601", recentSHA: "61caa2a50fa1792659fb71ab121515c6399f6fdcef98222c8ad59f6286d9b8e9", replacement: inotia2PatchBytes("11b50e4b1b68002b0fd0028992090c4bf8211888904203d004330139f9d104e058880849484301b010bd11bc08bc9e4670b505469bf6e4fc034b184700d42a0000d52a00e8030000d1ae1400")},
	{address: 0x2ada60, originalSHA: "cd00e292c5970d3c5e2f0ffa5171e555bc46bfc4faddfb4a418b6840b86e79a3", legacySHA: "cd00e292c5970d3c5e2f0ffa5171e555bc46bfc4faddfb4a418b6840b86e79a3", previousSHA: "cd00e292c5970d3c5e2f0ffa5171e555bc46bfc4faddfb4a418b6840b86e79a3", recentSHA: "cd00e292c5970d3c5e2f0ffa5171e555bc46bfc4faddfb4a418b6840b86e79a3", replacement: inotia2PatchBytes("70b5154c2068002823d0a069002819d06569002d11d0266a6868002806d091f693fe002802d1686899f67eff01202860002068600835013eeed12069002801d0064b1860064b1b68e068187000202060a06191f6f7fe70bd00d42a00f43419000c351900")},
	{address: 0x2ae580, originalSHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", legacySHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", previousSHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", recentSHA: "9d2122e123c700e22f4d62df0a39677eec91ba2666d7d2a689302a3c4b7f6796", replacement: inotia2PatchBytes("004b184761da2a000000000000000000000000000000000000000000000000000000000000000000")},
	{address: 0x2adb00, originalSHA: "f2c0d5456a983ecd12e314fcfa19879179fc8424343baeb1325457472ae85601", legacySHA: "f2c0d5456a983ecd12e314fcfa19879179fc8424343baeb1325457472ae85601", previousSHA: "f2c0d5456a983ecd12e314fcfa19879179fc8424343baeb1325457472ae85601", recentSHA: "f2c0d5456a983ecd12e314fcfa19879179fc8424343baeb1325457472ae85601", replacement: inotia2PatchBytes("104b1b68984205d0f0b506460b23f3560d4a104750b583b004460e46e37a019302ab009330466178a27801ab9df616fc002805d0019be372204600f021f8012003b050bd18d42a0085821600")},
	{address: 0x16827c, originalSHA: "229f02ecebc494c84883b1edc3fe434808b3f5cab2976e4188602d227efebf06", legacySHA: "229f02ecebc494c84883b1edc3fe434808b3f5cab2976e4188602d227efebf06", previousSHA: "229f02ecebc494c84883b1edc3fe434808b3f5cab2976e4188602d227efebf06", recentSHA: "229f02ecebc494c84883b1edc3fe434808b3f5cab2976e4188602d227efebf06", replacement: inotia2PatchBytes("004b184701db2a00")},
	{address: 0x2adb80, originalSHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", legacySHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", previousSHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", recentSHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", replacement: inotia2PatchBytes("70b50446e07a6178d6f614f90546a3799d4206d322799b18013b9d4202d9ad1a0135a57170bdc046")},
	{address: 0x2adc20, originalSHA: "6db65fd59fd356f6729140571b5bcd6bb3b83492a16e1bf0a3884442fc3c8a0e", legacySHA: "6db65fd59fd356f6729140571b5bcd6bb3b83492a16e1bf0a3884442fc3c8a0e", previousSHA: "6db65fd59fd356f6729140571b5bcd6bb3b83492a16e1bf0a3884442fc3c8a0e", recentSHA: "6db65fd59fd356f6729140571b5bcd6bb3b83492a16e1bf0a3884442fc3c8a0e", replacement: inotia2PatchBytes("064b1b68984201d10906090e00b5002901da002000bd024b1847c04618d42a00fb7e1600")},
	{address: 0x167ef0, originalSHA: "0e366ae56be176d4c11f684162e427fc08de75a5c1ff5a24386181e45410c947", legacySHA: "0e366ae56be176d4c11f684162e427fc08de75a5c1ff5a24386181e45410c947", previousSHA: "0e366ae56be176d4c11f684162e427fc08de75a5c1ff5a24386181e45410c947", recentSHA: "0e366ae56be176d4c11f684162e427fc08de75a5c1ff5a24386181e45410c947", replacement: inotia2PatchBytes("004b184721dc2a00")},
	{address: 0x2adc80, originalSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", legacySHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", previousSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", recentSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", replacement: inotia2PatchBytes("03b4044909688d42eb7a01d01b061b1603bc704718d42a00")},
	{address: 0x167f98, originalSHA: "52f4eb3d5bae3fdf1c827dfd86af7f16a1c882ef8188842274813815a3a04d2b", legacySHA: "52f4eb3d5bae3fdf1c827dfd86af7f16a1c882ef8188842274813815a3a04d2b", previousSHA: "52f4eb3d5bae3fdf1c827dfd86af7f16a1c882ef8188842274813815a3a04d2b", recentSHA: "52f4eb3d5bae3fdf1c827dfd86af7f16a1c882ef8188842274813815a3a04d2b", replacement: inotia2PatchBytes("45f172fe")},
	{address: 0x167fcc, originalSHA: "52f4eb3d5bae3fdf1c827dfd86af7f16a1c882ef8188842274813815a3a04d2b", legacySHA: "52f4eb3d5bae3fdf1c827dfd86af7f16a1c882ef8188842274813815a3a04d2b", previousSHA: "52f4eb3d5bae3fdf1c827dfd86af7f16a1c882ef8188842274813815a3a04d2b", recentSHA: "52f4eb3d5bae3fdf1c827dfd86af7f16a1c882ef8188842274813815a3a04d2b", replacement: inotia2PatchBytes("45f158fe")},
	{address: 0x2adcb0, originalSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", legacySHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", previousSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", recentSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", replacement: inotia2PatchBytes("03b4044909688842c37a01d01b061b1603bc704718d42a00")},
	{address: 0x120690, originalSHA: "9e0783b5746c142860ab0416e7a4e9b06984b582570168c1ea59202ccc4029af", legacySHA: "9e0783b5746c142860ab0416e7a4e9b06984b582570168c1ea59202ccc4029af", previousSHA: "9e0783b5746c142860ab0416e7a4e9b06984b582570168c1ea59202ccc4029af", recentSHA: "9e0783b5746c142860ab0416e7a4e9b06984b582570168c1ea59202ccc4029af", replacement: inotia2PatchBytes("8df10efb")},
	{address: 0x2adce0, originalSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", legacySHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", previousSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", recentSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", replacement: inotia2PatchBytes("03b4044909688e42f37a01d01b061b1603bc704718d42a00")},
	{address: 0x120404, originalSHA: "d5a4839efd1ee564a9459aa16456af8f1f062c7612cc72e3a614af821b917d72", legacySHA: "d5a4839efd1ee564a9459aa16456af8f1f062c7612cc72e3a614af821b917d72", previousSHA: "d5a4839efd1ee564a9459aa16456af8f1f062c7612cc72e3a614af821b917d72", recentSHA: "d5a4839efd1ee564a9459aa16456af8f1f062c7612cc72e3a614af821b917d72", replacement: inotia2PatchBytes("8df16cfc")},
	{address: 0x1682a0, originalSHA: "d5a4839efd1ee564a9459aa16456af8f1f062c7612cc72e3a614af821b917d72", legacySHA: "d5a4839efd1ee564a9459aa16456af8f1f062c7612cc72e3a614af821b917d72", previousSHA: "d5a4839efd1ee564a9459aa16456af8f1f062c7612cc72e3a614af821b917d72", recentSHA: "d5a4839efd1ee564a9459aa16456af8f1f062c7612cc72e3a614af821b917d72", replacement: inotia2PatchBytes("45f11efd")},
	{address: 0x2add10, originalSHA: "17b0761f87b081d5cf10757ccc89f12be355c70e2e29df288b65b30710dcbcd1", legacySHA: "17b0761f87b081d5cf10757ccc89f12be355c70e2e29df288b65b30710dcbcd1", previousSHA: "17b0761f87b081d5cf10757ccc89f12be355c70e2e29df288b65b30710dcbcd1", recentSHA: "17b0761f87b081d5cf10757ccc89f12be355c70e2e29df288b65b30710dcbcd1", replacement: inotia2PatchBytes("74d82a001b0000000500000000d52a00440000000c00000010d62a00840000001600000020d82a001500000004000000")},
	{address: 0x2add50, originalSHA: "81c611f35bff79491538b2f7cf201c7597a661a5c549633541c62bdc8af1613f", legacySHA: "81c611f35bff79491538b2f7cf201c7597a661a5c549633541c62bdc8af1613f", previousSHA: "81c611f35bff79491538b2f7cf201c7597a661a5c549633541c62bdc8af1613f", recentSHA: "81c611f35bff79491538b2f7cf201c7597a661a5c549633541c62bdc8af1613f", replacement: inotia2PatchBytes("10b5baf65df8baf609fa0446002c2ed091f6a0fd164b1b681878164a106001201870012060700420a0700120e070042020718420e071142020720020e072042020730d4820610d48e0610d4b1b681c600c4b0020186071f673ff002000210a4b1a68b6f6b7fe00200849bbf685f810bd0c3519000cd42a0060df2a006d0612005c2a190018d42a0078aa2a0080e72a00")},
	{address: 0x2ade20, originalSHA: "85ada57e1f601e962d705f389285adb4e74f450bc00672240dfef7399d82457f", legacySHA: "85ada57e1f601e962d705f389285adb4e74f450bc00672240dfef7399d82457f", previousSHA: "85ada57e1f601e962d705f389285adb4e74f450bc00672240dfef7399d82457f", recentSHA: "85ada57e1f601e962d705f389285adb4e74f450bc00672240dfef7399d82457f", replacement: inotia2PatchBytes("70b50446204d2868002832d01f4b1e68b06c84421cd0a86800282ad03069844206d01b4b1b6818682146baf617fa2ae0174b1b681b68d87a032824d8686072f62ff8012028600020a860fff739fd1ae0ae6872f625f8002e05d101202860a860fff766ff0fe00020a8600220c9f6a4f809e0204670bc08bc9e46f0b557464646c0b4044b184770bd00d42a00e82819005c2a1900a1031200")},
	{address: 0x120398, originalSHA: "8af9f2d5cf390b08711473c72ecd6d3816cfff1d92e31a6b9c56c655874554aa", legacySHA: "8af9f2d5cf390b08711473c72ecd6d3816cfff1d92e31a6b9c56c655874554aa", previousSHA: "8af9f2d5cf390b08711473c72ecd6d3816cfff1d92e31a6b9c56c655874554aa", recentSHA: "8af9f2d5cf390b08711473c72ecd6d3816cfff1d92e31a6b9c56c655874554aa", replacement: inotia2PatchBytes("004b184721de2a00")},
	{address: 0x2adf10, originalSHA: "5dcc1b5872dd9ff1c234501f1fefda01f664164e1583c3e1bb3dbea47588ab31", legacySHA: "5dcc1b5872dd9ff1c234501f1fefda01f664164e1583c3e1bb3dbea47588ab31", previousSHA: "5dcc1b5872dd9ff1c234501f1fefda01f664164e1583c3e1bb3dbea47588ab31", recentSHA: "5dcc1b5872dd9ff1c234501f1fefda01f664164e1583c3e1bb3dbea47588ab31", replacement: inotia2PatchBytes("084a1268012a07d1074bc21a032a03d89200064b9858704700b5054a054b06490847c04608d42a006113000080df2a00c42419006017000015fd1400")},
	{address: 0x2adf60, originalSHA: "374708fff7719dd5979ec875d56cd2286f6d3cf7ec317a3b25632aab28ec37bb", legacySHA: "374708fff7719dd5979ec875d56cd2286f6d3cf7ec317a3b25632aab28ec37bb", previousSHA: "374708fff7719dd5979ec875d56cd2286f6d3cf7ec317a3b25632aab28ec37bb", recentSHA: "374708fff7719dd5979ec875d56cd2286f6d3cf7ec317a3b25632aab28ec37bb", replacement: inotia2PatchBytes("1c0000001d0000001e0000001f000000")},
	{address: 0x2adf80, originalSHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", legacySHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", previousSHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", recentSHA: "2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", replacement: inotia2PatchBytes("90df2a0097df2a009cdf2a00a1df2a00bcd2baf1c5db00b9abb1e200c0e5baf100c0e5bdc5b1b800")},
	{address: 0x2ad500, originalSHA: "5f51d9e57c051cc516388346d860976bff3531da9561f6570c19b2ca2da48eab", legacySHA: "5f51d9e57c051cc516388346d860976bff3531da9561f6570c19b2ca2da48eab", previousSHA: "5f51d9e57c051cc516388346d860976bff3531da9561f6570c19b2ca2da48eab", recentSHA: "5f51d9e57c051cc516388346d860976bff3531da9561f6570c19b2ca2da48eab", replacement: inotia2PatchBytes("d5020a00f8010a00aa030a00f7010a00f9010a00a9030a009a030a00f8020a0043020a0044020a00e9020a009c020a009b030a00f3020a00ab030a00ac030a00dd020a00d1020a009c030a00e1020a009d030a00e5020a00ed020a00d6020a009e030a00da020a00e6020a00f0020a00d7020a009f030a00a0030a00a1030a00a2030a00d2020a00d3020a00d4020a00d8020a00d9020a00ad030a00db020a00dc020a00de020a00df020a00e0020a00e2020a00e3020a00e4020a00e7020a00e8020a00ea020a00eb020a00ec020a00ee020a00ef020a00f1020a00f2020a00f4020a00f5020a00f6020a00f7020a00f9020a00fa020a00ba030a00b9030a00b8030a00bc030a00bd030a00bb030a00b4030a00b6030a00b7030a00fb020a0003030a00bf020a0071030a0008030a0087030a00b3020a0009020a00c9020a0077030a000d030a009e020a00bb020a007c030a00c0020a008b030a0012030a0082030a00ac020a009f020a0072030a00b4020a0017030a0078030a00a0020a00c1020a001c030a007d030a00ad020a00a1020a0097030a00b5020a0083030a00ca020a0088030a0073030a008c030a00a2020a0079030a00bc020a007e030a00c2020a00a3020a00ae020a00ff020a00b6020a0004030a0084030a00a4020a0099030a0009030a00c3020a0074030a00a5020a00af020a0089030a000e030a00b7020a00cb020a00a6020a0013030a00bd020a00c4020a007a030a00a7020a00b0020a0018030a008d030a00b8020a001d030a007f030a00a8020a00c5020a0085030a00b1020a00a9020a00b9020a00cc020a0075030a0080030a00aa020a0076030a007b030a0081030a0086030a00c6020a008a030a008e030a0098030a0005030a0006030a0007030a003d030a000a030a000b030a000c030a000f030a0010030a0011030a0014030a0015030a0016030a00c7020a00c8020a00ab020a00b2020a00ba020a00be020a00cd020a00ce020a0019030a001a030a001b030a001e030a001f030a0020030a00fc020a00fd020a00fe020a0000030a0001030a0002030a00c3030a00c2030a00c6030a00be030a00bf030a00c0030a00c1030a004e020a008f030a0093030a0021030a0090030a0026030a0094030a0091030a0022030a0095030a0027030a0092030a0096030a0023030a0024030a0025030a0028030a0029030a002a030a00c4030a00c5030a0059030a0099020a004303050044030a0047030a00af031400b0031400b1031400b2031400b3031400b50314008702050045030a0046030a0004001e00a5033200a6036400a7031e00a8031e0070030a002c030a00cc030a00ce030a00ca030a00cb030a00cd030a00cf030a00")},
	{address: 0x2ae780, originalSHA: "5322fecfc92a5e3248a297a3df3eddfb9bd9049504272e4f572b87fa36d4b3bd", legacySHA: "5322fecfc92a5e3248a297a3df3eddfb9bd9049504272e4f572b87fa36d4b3bd", previousSHA: "97274784603480cd2b1d9e98f8eb0dc166181ef0029ea6231202c2aa106c1466", recentSHA: "97274784603480cd2b1d9e98f8eb0dc166181ef0029ea6231202c2aa106c1466", replacement: inotia2PatchBytes("b7cec4c320b0f1b5e520bbf3c1a100")},
	{address: 0x120890, originalSHA: "c26fbefa97ddecbc7bdff67dfeb4021ce6c213256be0c909c8ab2bcf22959881", legacySHA: "47d0a4b5c5147e4b360d558c6a6473e5335a9bff6b59f608c313d4e666fd49dc", previousSHA: "47d0a4b5c5147e4b360d558c6a6473e5335a9bff6b59f608c313d4e666fd49dc", recentSHA: "47d0a4b5c5147e4b360d558c6a6473e5335a9bff6b59f608c313d4e666fd49dc", replacement: inotia2PatchBytes("004b184701e22a00")},
	{address: 0x1205a8, originalSHA: "d440ac40c5248ac74e645983fad8b8ddeaa920da77e77f151468db0cb8d6178a", legacySHA: "10a5bd418c183a937e381b3b8ff561d049a1bf10fc1671d4a037093c14797dfe", previousSHA: "10a5bd418c183a937e381b3b8ff561d049a1bf10fc1671d4a037093c14797dfe", recentSHA: "10a5bd418c183a937e381b3b8ff561d049a1bf10fc1671d4a037093c14797dfe", replacement: inotia2PatchBytes("004b184741e22a00")},
	{address: 0x14aec8, originalSHA: "6d2c56da87762ababfdd04e2c1d374784eb4ba3c43111517bd1344d2777aabfe", legacySHA: "c7b84bb11ede0f2062814d98652157a9d3ec25ebc1ba271391384dc60224d0d7", previousSHA: "c7b84bb11ede0f2062814d98652157a9d3ec25ebc1ba271391384dc60224d0d7", recentSHA: "c7b84bb11ede0f2062814d98652157a9d3ec25ebc1ba271391384dc60224d0d7", replacement: inotia2PatchBytes("004b184701e52a00")},
	{address: 0x11fec8, originalSHA: "89a28ca5bfe82a53418cd53957380216b65b694a38d206603d5753aef8866674", legacySHA: "93edb1a3cc9fc5cc6faa894bad1e2b7363acfa40ebc32f923e740f8e1c3b92f6", previousSHA: "93edb1a3cc9fc5cc6faa894bad1e2b7363acfa40ebc32f923e740f8e1c3b92f6", recentSHA: "93edb1a3cc9fc5cc6faa894bad1e2b7363acfa40ebc32f923e740f8e1c3b92f6", replacement: inotia2PatchBytes("8ef15afb")},
}

func installInotia2OfflineShop(client []byte, backend cpu.Backend) error {
	// Leave package bytes, title/save hashes, BSS bootstrap size and audio/frame
	// behavior at the v1008 baseline. Unknown games never reach a memory write.
	if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA {
		return nil
	}
	return installInotia2OfflineShopPatches(backend, inotia2OfflineShopPatches)
}

func installInotia2OfflineShopPatches(backend cpu.Backend, patches []inotia2OfflineShopPatch) error {
	originalState, legacyState, previousState, recentState, installedState := true, true, true, true, true
	for i, p := range patches {
		if len(p.replacement) == 0 || uint64(p.address)+uint64(len(p.replacement)) > 1<<32 {
			return fmt.Errorf("Inotia2 local shop patch %d has invalid bounds", i)
		}
		for _, q := range patches[:i] {
			if uint64(p.address) < uint64(q.address)+uint64(len(q.replacement)) && uint64(q.address) < uint64(p.address)+uint64(len(p.replacement)) {
				return fmt.Errorf("Inotia2 local shop patch %d overlaps", i)
			}
		}
		before := make([]byte, len(p.replacement))
		if err := backend.ReadMemory(p.address, before); err != nil {
			return err
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(before))
		original := digest == p.originalSHA
		legacy := p.legacySHA != "" && digest == p.legacySHA
		previous := p.previousSHA != "" && digest == p.previousSHA
		recent := p.recentSHA != "" && digest == p.recentSHA
		installed := bytes.Equal(before, p.replacement)
		if !original && !legacy && !previous && !recent && !installed {
			return fmt.Errorf("Inotia2 local shop patch %d original checksum mismatch", i)
		}
		originalState = originalState && original
		legacyState = legacyState && legacy
		previousState = previousState && previous
		recentState = recentState && recent
		installedState = installedState && installed
	}
	if installedState {
		return nil
	}
	if !originalState && !legacyState && !previousState && !recentState {
		return fmt.Errorf("Inotia2 local shop patch installation is incomplete")
	}
	// Validate every span before changing code. Execution has not started here;
	// accept a complete known original or previous image, never a mixture.
	// Restore older full machine states with the current catalog.
	// An unexpected mapped-memory write error aborts loading the title.
	for _, p := range patches {
		if err := backend.WriteMemory(p.address, p.replacement); err != nil {
			return err
		}
	}
	return nil
}
