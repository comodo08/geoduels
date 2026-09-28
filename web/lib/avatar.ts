export type AvatarKind = "player" | "team-red" | "team-blue";
export const defaultAvatars: Record<AvatarKind,string> = {
 player: "/avatars/player.v1.webp", "team-red": "/avatars/team-red.v1.webp", "team-blue": "/avatars/team-blue.v1.webp",
};
export function avatarSource(url?: string, kind: AvatarKind = "player") {
 const value=url?.trim();
 if(value?.startsWith("/")&&!value.startsWith("//"))return value;
 if(value){try{const parsed=new URL(value);if(parsed.protocol==="https:"||parsed.protocol==="http:")return value;}catch{}}
 return defaultAvatars[kind];
}
// DOM-based markers and React avatars use the same assets and failure policy.
export function avatarImage(url?:string,kind:AvatarKind="player") {
 const image=document.createElement("img");image.src=avatarSource(url,kind);image.alt="Player avatar";
 image.addEventListener("error",()=>{image.src=defaultAvatars[kind];},{once:true});return image;
}
