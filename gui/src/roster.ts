// This file shows a possible roster configuration, which uses the production
// blockchain. This template is used for the deployment.
export function getRosterStr () {
  return `[[servers]]
  Address = "tls://localhost:7774"
  Suite = "Ed25519"
  Public = "b6e760694d96649c5b384dd48bd64a034fe184ebb586f1251293abcb8c02fe94"
  Description = "Conode_3"
  [servers.Services]
    [servers.Services.ByzCoin]
      Public = "677c9417c3268fce456975995386f530bc70ded13530f732f162d26fec24433f105fc70ec478341e60af55f28f03490e4b99fbfa3407143457c6a8a9213877715897ffd98f71be9684610a35ffde77eab738171ad9204eee629262a25b8ece145630ca44b9b07f40aef23ab7426c8397eb7ceb458c6e794ea44d7c2b6d30da2e"
      Suite = "bn256.adapter"
    [servers.Services.Skipchain]
      Public = "53cc4fc7944657c7babe39abca200f1a2b3538f0b5b07484019ff94fc2ce6b984534d3cdea7a75c4dd0a16b323bc093f0f82020db0c2feef36001a03eeb7c9051a9e0fbcfb63725add760638c6ec8f7c64a4c3ed1cc833ac9801c3b207dcdf061dde7d4103adbdb5f9f55deb22c6a5915155ad8281973104ae415d2d5e3abeea"
      Suite = "bn256.adapter"
[[servers]]
  Address = "tls://localhost:7772"
  Suite = "Ed25519"
  Public = "b8de51070f0b317dd4b1cf29ec02a18f027a889b663c2d3456fcd52321e55632"
  Description = "Conode_2"
  [servers.Services]
    [servers.Services.ByzCoin]
      Public = "3f552ae4ddd20c8bb8b35a034d7576ce6b7ecb31c0a6a1c37f0af542db576dcb576f49c2cd8814ccca2165a61341fd43a4c380bd7dfe87625604ca4ec22dbf8b379fce5fa06ea0ae7f1c4d7c734dabdda0f29a0deed0ff54de473e96e5b038831daf29d5777d815ecda3afed94c993a9ee8c17637204e673ea3f6b50ebe07445"
      Suite = "bn256.adapter"
    [servers.Services.Skipchain]
      Public = "5efa159db63adfd5d70fdc5ef4d5dcce4023cabeb630b3e855b299fcf477329b63f3dbfd15978b2a660d79125c071671f1001804af3ad9ed48e803381bee29446e37b9fc5f183a34b0fda4dd75aff63a78ccc521b7e2f6b35b03d7dd8f3091074cc7c440c706d47ab88665e045791967d987a435b800abe21fd06d8c6b2818c3"
      Suite = "bn256.adapter"
[[servers]]
  Address = "tls://localhost:7770"
  Suite = "Ed25519"
  Public = "4916678de3b455381592b5743dd44dfc52b263b8ba20739f063c7ec8aa828c49"
  Description = "Conode_1"
  [servers.Services]
    [servers.Services.ByzCoin]
      Public = "3ea58f70a7c057f9d47e76cf11e155d21116ba6bcd2b2de16188c7e4c1b173648e6d8a927c3efdb72394a58bc00ed06109c36390571e5858563c0f5a4b7073e35acbe96d647b04f587c5dc8d524e8d29af86170b794fb5278bae13370b73f968288d554047d87a7ce19d85b2a8226759efda25f81cbf84759039c1d68dec9ffc"
      Suite = "bn256.adapter"
    [servers.Services.Skipchain]
      Public = "35ee503dbebef7e4fa7e8287f28fef8ce8a0909b06010e16013261fcd4368b4c04945b6452501e4c19060493873f62b43840b736827d2e72cc3fdcabc5b27eb1397271d826c78cb5e5e6f16dc6ce97bc46bf34534aaaa870823832c17afa3b694d28d3bee2fc668a3f53c95072158157f3135f597abc49389c76bed7377ef899"
      Suite = "bn256.adapter"`
}
