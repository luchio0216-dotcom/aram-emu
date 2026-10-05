package application

import (
	"crypto/sha256"
	"fmt"
	"github.com/mirusu400/aram-core/cpu"
)

const inotia2SkillBooksEnsure = uint32(0x2aea40)
const inotia2SkillBooksUse = uint32(0x2aeb60)
const inotia2SkillBooksName = uint32(0x2aec10)
const inotia2SkillBooksMenu = uint32(0x2aef00)

// Six distinct native item IDs retain normal purchase, stack, discard and save
// serialization. Only the known client's startup and item routines are hooked.
// Definitions are copied from its own sealed book at runtime, never embedded.
var inotia2SkillBooksPatches = []inotia2OfflineShopPatch{
	{address: 0x2aea40, originalSHA: "6edd9f6f9cc92cded36e6c4a580933f9c9f1b90562b46903b806f21902a1a54f", replacement: inotia2PatchBytes("f0b5154c2088154b984223d1144d2e68002e1fd0134877f6fdf800281ad007463146114ad5f67ef8104bf6180e4bfb181c460625204631461122d5f673f80c4b5b1b23801134013df4d1054b1f60024b08481880f0bdc04624481900ca03000028481900d04000006a400000b737000066ea0000d0030000")},
	{address: 0x2aeb00, originalSHA: "374708fff7719dd5979ec875d56cd2286f6d3cf7ec317a3b25632aab28ec37bb", replacement: inotia2PatchBytes("10b552f625ff0fb4fff79aff0fbc10bd")},
	{address: 0x2aeb20, originalSHA: "3addfb141cd7c9c4c6543a82191a3707ac29c7a041217782e61d4d91c691aee8", replacement: inotia2PatchBytes("0fb5fff78dff0fbc08bc9e46f0b557464646c0b4004b1847c5a01400")},
	{address: 0x2aeb60, originalSHA: "115bad14f1c9f2c027a84de21b107015722cb76be8d0abf3760ad8e00d6c24a5", replacement: inotia2PatchBytes("70b504461a4be41a052c27d819489bf6a5fa0546002d19d000200b2177f61af8062800d3023023011b18284600211d2298f6f0ff002806d0284698f675fd002801d0012070bd284698f6eefe0820002100220023baf65afe002070bd70bc08bc9e4630b5044b054dc018054b1847c046ca0300006e010000d3fcffff907f040031a51400")},
	{address: 0x2aec10, originalSHA: "d4817aa5497628e7c77e6b606107042bbba3130888c5f47a375e6179be789fbb", replacement: inotia2PatchBytes("04b5094bc21a052a06d8002900d006329200064b985804bd04bc08bc9e4670b5034c0906034b1847ca03000080ec2a00c4241900a58c1400")},
	{address: 0x2aec50, originalSHA: "17b0761f87b081d5cf10757ccc89f12be355c70e2e29df288b65b30710dcbcd1", replacement: inotia2PatchBytes("064bc21a052a03d89200054b9858704700b5044a044b05490847c04660ea000080ec2a00c42419006017000015fd1400")},
	{address: 0x2aec80, originalSHA: "17b0761f87b081d5cf10757ccc89f12be355c70e2e29df288b65b30710dcbcd1", replacement: inotia2PatchBytes("00ed2a0018ed2a002eed2a0042ed2a005ced2a0074ed2a008eed2a00c3ed2a00f6ed2a0027ee2a005eee2a0093ee2a00")},
	{address: 0x2aed00, originalSHA: "3b9e9d4ed135268270adc99df1f1ceb9f5df2752e3e5c20ae5b42690d5113e16", replacement: inotia2PatchBytes("bac0c0ceb5c820bdbac5b3bacf28b9d9b9d9b8aebec82900bac0c0ceb5c820bdbac5b3bacf28c5dbc7c3b7af2900bac0c0ceb5c820bdbac5b3bacf28b7ceb1d72900bac0c0ceb5c820bdbac5b3bacf28bda6b5b5bfecc7e5c5cd2900bac0c0ceb5c820bdbac5b3bacf28c7c1b8aebdbac6ae2900bac0c0ceb5c820bdbac5b3bacf28bec6c5a9b8dec0ccc1f62900bbe7bfebc7cfb8e920b9d9b9d9b8aebec8c0c720bdbac5b3bacf20c7cfb3aab8a620b9abc0dbc0a7b7ce20bef2bdc0b4cfb4d92e00bbe7bfebc7cfb8e920c5dbc7c3b7afc0c720bdbac5b3bacf20c7cfb3aab8a620b9abc0dbc0a7b7ce20bef2bdc0b4cfb4d92e00bbe7bfebc7cfb8e920b7ceb1d7c0c720bdbac5b3bacf20c7cfb3aab8a620b9abc0dbc0a7b7ce20bef2bdc0b4cfb4d92e00bbe7bfebc7cfb8e920bda6b5b5bfecc7e5c5cdc0c720bdbac5b3bacf20c7cfb3aab8a620b9abc0dbc0a7b7ce20bef2bdc0b4cfb4d92e00bbe7bfebc7cfb8e920c7c1b8aebdbac6aec0c720bdbac5b3bacf20c7cfb3aab8a620b9abc0dbc0a7b7ce20bef2bdc0b4cfb4d92e00bbe7bfebc7cfb8e920bec6c5a9b8dec0ccc1f6c0c720bdbac5b3bacf20c7cfb3aab8a620b9abc0dbc0a7b7ce20bef2bdc0b4cfb4d92e00")},
	{address: 0x1450be, originalSHA: "579a8982479736d76fe026604741b0cddd2f17d576bf39385cdcaae3f50a09fa", replacement: inotia2PatchBytes("69f11ffd")},
	{address: 0x14a0bc, originalSHA: "8af9f2d5cf390b08711473c72ecd6d3816cfff1d92e31a6b9c56c655874554aa", replacement: inotia2PatchBytes("004b184721eb2a00")},
	{address: 0x14a528, originalSHA: "c140c02cff6a1f7320d4bdfa0b2818427f330092a545b8fc3091b2e374b5b238", replacement: inotia2PatchBytes("004b184761eb2a00")},
	{address: 0x148c9c, originalSHA: "d32070d91eb80ac3b5f0badec7a69dd4cdc21d0411bcafbcb4438887dd277e1d", replacement: inotia2PatchBytes("004b184711ec2a00")},
	{address: 0x14fd0c, originalSHA: "4aa78371501c1169e4cc6162102409dac9be79ccdd061d01835acc34f6beddbf", replacement: inotia2PatchBytes("004b184751ec2a00")},
}

// Install separately so full states saved with the v1017 item hooks can receive
// the new inventory-menu hook without being rejected as a partial installation.
var inotia2SkillBooksMenuPatches = []inotia2OfflineShopPatch{
	{address: 0x2aef00, originalSHA: "9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0", replacement: inotia2PatchBytes("034bc21a052a01d801207047014b1847ca030000197a1400")},
	{address: 0x111422, originalSHA: "7575fdb9b91fbc20da01a6c12e27a23b2aae3c8886c0d9b328da4414f21be8b3", replacement: inotia2PatchBytes("9df16dfd")},
}

func installInotia2SkillBooks(client []byte, backend cpu.Backend) error {
	if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA {
		return nil
	}
	if err := installInotia2OfflineShopPatches(backend, inotia2SkillBooksPatches); err != nil {
		return fmt.Errorf("Inotia2 class sealed skillbooks: %w", err)
	}
	if err := installInotia2OfflineShopPatches(backend, inotia2SkillBooksMenuPatches); err != nil {
		return fmt.Errorf("Inotia2 class sealed skillbooks inventory menu: %w", err)
	}
	return nil
}
