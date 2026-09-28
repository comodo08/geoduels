import sharp from "sharp";
import {mkdir,writeFile} from "node:fs/promises";
import {fileURLToPath} from "node:url";
const root=fileURLToPath(new URL("../",import.meta.url));
await mkdir(`${root}public/avatars`,{recursive:true});
const user='<circle cx="64" cy="47" r="16"/><path d="M34 101v-8c0-17 13-27 30-27s30 10 30 27v8"/>';
const pin='<path d="M64 108S32 76 32 50a32 32 0 0 1 64 0c0 26-32 58-32 58Z"/><circle cx="64" cy="50" r="11" fill="none" stroke-width="7"/>';
for(const [name,color,art] of [['player','#10b981',user],['team-red','#ef4444',pin],['team-blue','#3b82f6',pin]]){
 const svg=`<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128" viewBox="0 0 128 128"><rect width="128" height="128" rx="64" fill="${color}"/><g fill="white" stroke="${color}" stroke-width="5" stroke-linejoin="round">${art}</g></svg>`;
 await sharp(Buffer.from(svg)).webp({lossless:true,effort:6}).toFile(`${root}public/avatars/${name}.v1.webp`);
}
await sharp(`${root}public/badges/map-winner-badge.v1.png`).resize(256,256,{fit:"inside",withoutEnlargement:true}).webp({quality:90,effort:6}).toFile(`${root}public/badges/map-winner-badge.v1.webp`);
