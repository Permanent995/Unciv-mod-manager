export namespace app {
	
	export class AppConfig {
	    uncivPath: string;
	    savedPaths: string[];
	    lastActivePath: string;
	    zoomLevel: number;
	    sidebarPos: string;
	    sidebarWidth: number;
	    hiddenNav: string[];
	    theme: string;
	    translateProvider: string;
	    translateCustomUrl: string;
	    translateCustomKey: string;
	    translateCustomModel: string;
	    githubToken: string;
	    mpServer: string;
	    mpUid: string;
	    mpPassword: string;
	    customMirrors: string[];
	    maxSaves: number;
	    mirrorMode: string;
	    selectedMirror: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uncivPath = source["uncivPath"];
	        this.savedPaths = source["savedPaths"];
	        this.lastActivePath = source["lastActivePath"];
	        this.zoomLevel = source["zoomLevel"];
	        this.sidebarPos = source["sidebarPos"];
	        this.sidebarWidth = source["sidebarWidth"];
	        this.hiddenNav = source["hiddenNav"];
	        this.theme = source["theme"];
	        this.translateProvider = source["translateProvider"];
	        this.translateCustomUrl = source["translateCustomUrl"];
	        this.translateCustomKey = source["translateCustomKey"];
	        this.translateCustomModel = source["translateCustomModel"];
	        this.githubToken = source["githubToken"];
	        this.mpServer = source["mpServer"];
	        this.mpUid = source["mpUid"];
	        this.mpPassword = source["mpPassword"];
	        this.customMirrors = source["customMirrors"];
	        this.maxSaves = source["maxSaves"];
	        this.mirrorMode = source["mirrorMode"];
	        this.selectedMirror = source["selectedMirror"];
	    }
	}
	export class ConflictReport {
	    level: string;
	    category: string;
	    modA: string;
	    modB: string;
	    entityID: string;
	    message: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new ConflictReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.category = source["category"];
	        this.modA = source["modA"];
	        this.modB = source["modB"];
	        this.entityID = source["entityID"];
	        this.message = source["message"];
	        this.detail = source["detail"];
	    }
	}
	export class CrashInfo {
	    found: boolean;
	    filePath: string;
	    lastModTime: string;
	    raw: string;
	    diagnosis: string;
	    suggestion: string;
	    hasMatch: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CrashInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.found = source["found"];
	        this.filePath = source["filePath"];
	        this.lastModTime = source["lastModTime"];
	        this.raw = source["raw"];
	        this.diagnosis = source["diagnosis"];
	        this.suggestion = source["suggestion"];
	        this.hasMatch = source["hasMatch"];
	    }
	}
	export class DiagIssue {
	    mod: string;
	    severity: string;
	    message: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new DiagIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mod = source["mod"];
	        this.severity = source["severity"];
	        this.message = source["message"];
	        this.detail = source["detail"];
	    }
	}
	export class DownloadTask {
	    id: string;
	    url: string;
	    filename: string;
	    status: string;
	    totalSize: number;
	    downloaded: number;
	    percent: number;
	    speed: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.status = source["status"];
	        this.totalSize = source["totalSize"];
	        this.downloaded = source["downloaded"];
	        this.percent = source["percent"];
	        this.speed = source["speed"];
	        this.error = source["error"];
	    }
	}
	export class GHAsset {
	    name: string;
	    size: number;
	    browser_download_url: string;
	
	    static createFrom(source: any = {}) {
	        return new GHAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	        this.browser_download_url = source["browser_download_url"];
	    }
	}
	export class GHRelease {
	    tag_name: string;
	    name: string;
	    published_at: string;
	    zipball_url: string;
	    assets: GHAsset[];
	
	    static createFrom(source: any = {}) {
	        return new GHRelease(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tag_name = source["tag_name"];
	        this.name = source["name"];
	        this.published_at = source["published_at"];
	        this.zipball_url = source["zipball_url"];
	        this.assets = this.convertValues(source["assets"], GHAsset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MapInfo {
	    name: string;
	    path: string;
	    source: string;
	    modFolder?: string;
	
	    static createFrom(source: any = {}) {
	        return new MapInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.source = source["source"];
	        this.modFolder = source["modFolder"];
	    }
	}
	export class MirrorInfo {
	    url: string;
	    label: string;
	    latency: number;
	    alive: boolean;
	    isCustom: boolean;
	    lastChecked: string;
	
	    static createFrom(source: any = {}) {
	        return new MirrorInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.label = source["label"];
	        this.latency = source["latency"];
	        this.alive = source["alive"];
	        this.isCustom = source["isCustom"];
	        this.lastChecked = source["lastChecked"];
	    }
	}
	export class ModBackup {
	    folder: string;
	    timestamp: string;
	    version: string;
	    path: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new ModBackup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.timestamp = source["timestamp"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.size = source["size"];
	    }
	}
	export class ModInfo {
	    name: string;
	    folder: string;
	    author?: string;
	    isBaseRuleset: boolean;
	    topics?: string[];
	    modUrl?: string;
	    lastUpdated?: string;
	    modSize: number;
	    category: string;
	    isIncomplete: boolean;
	    hasReadme: boolean;
	    hasPreview: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.folder = source["folder"];
	        this.author = source["author"];
	        this.isBaseRuleset = source["isBaseRuleset"];
	        this.topics = source["topics"];
	        this.modUrl = source["modUrl"];
	        this.lastUpdated = source["lastUpdated"];
	        this.modSize = source["modSize"];
	        this.category = source["category"];
	        this.isIncomplete = source["isIncomplete"];
	        this.hasReadme = source["hasReadme"];
	        this.hasPreview = source["hasPreview"];
	    }
	}
	export class ModUpdateInfo {
	    folder: string;
	    name: string;
	    currentVer: string;
	    latestVer: string;
	    modUrl: string;
	    hasUpdate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModUpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.name = source["name"];
	        this.currentVer = source["currentVer"];
	        this.latestVer = source["latestVer"];
	        this.modUrl = source["modUrl"];
	        this.hasUpdate = source["hasUpdate"];
	    }
	}
	export class MultiplayerDiff {
	    mod: string;
	    issue: string;
	    valueA: string;
	    valueB: string;
	
	    static createFrom(source: any = {}) {
	        return new MultiplayerDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mod = source["mod"];
	        this.issue = source["issue"];
	        this.valueA = source["valueA"];
	        this.valueB = source["valueB"];
	    }
	}
	export class OnlineMod {
	    name: string;
	    owner: string;
	    repo: string;
	    description: string;
	    stars: number;
	    updatedAt: string;
	    topics: string[];
	    htmlUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new OnlineMod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.owner = source["owner"];
	        this.repo = source["repo"];
	        this.description = source["description"];
	        this.stars = source["stars"];
	        this.updatedAt = source["updatedAt"];
	        this.topics = source["topics"];
	        this.htmlUrl = source["htmlUrl"];
	    }
	}
	export class ProxyConfig {
	    mode: string;
	    mirrorUrl: string;
	    customProxy: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.mirrorUrl = source["mirrorUrl"];
	        this.customProxy = source["customProxy"];
	    }
	}
	export class SaveArchive {
	    name: string;
	    origName: string;
	    timestamp: string;
	    path: string;
	    fileSize: number;
	    modifiedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveArchive(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.origName = source["origName"];
	        this.timestamp = source["timestamp"];
	        this.path = source["path"];
	        this.fileSize = source["fileSize"];
	        this.modifiedAt = source["modifiedAt"];
	    }
	}
	export class SaveInfo {
	    name: string;
	    path: string;
	    fileSize: number;
	    modifiedAt: string;
	    civName?: string;
	    turn?: number;
	    version?: string;
	    mods?: string[];
	
	    static createFrom(source: any = {}) {
	        return new SaveInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.fileSize = source["fileSize"];
	        this.modifiedAt = source["modifiedAt"];
	        this.civName = source["civName"];
	        this.turn = source["turn"];
	        this.version = source["version"];
	        this.mods = source["mods"];
	    }
	}
	export class SelfUpdateInfo {
	    currentVersion: string;
	    latestVersion: string;
	    downloadUrl: string;
	    hasUpdate: boolean;
	    releaseName: string;
	    cachedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new SelfUpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.downloadUrl = source["downloadUrl"];
	        this.hasUpdate = source["hasUpdate"];
	        this.releaseName = source["releaseName"];
	        this.cachedAt = source["cachedAt"];
	    }
	}
	export class UncivInfo {
	    hasExe: boolean;
	    hasJar: boolean;
	    hasModsDir: boolean;
	    hasSettings: boolean;
	    isValid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UncivInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasExe = source["hasExe"];
	        this.hasJar = source["hasJar"];
	        this.hasModsDir = source["hasModsDir"];
	        this.hasSettings = source["hasSettings"];
	        this.isValid = source["isValid"];
	    }
	}
	export class UncivPathOption {
	    path: string;
	    version: string;
	    hasExe: boolean;
	    hasJar: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UncivPathOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.version = source["version"];
	        this.hasExe = source["hasExe"];
	        this.hasJar = source["hasJar"];
	    }
	}

}

export namespace main {
	
	export class DocInfo {
	    name: string;
	    title: string;
	    size: number;
	    modTime: string;
	
	    static createFrom(source: any = {}) {
	        return new DocInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	    }
	}

}

