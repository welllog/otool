export namespace srvs {
	
	export class ImageFile {
	    name: string;
	    type: string;
	    body: number[];
	
	    static createFrom(source: any = {}) {
	        return new ImageFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.body = source["body"];
	    }
	}
	export class ImageInfo {
	    name: string;
	    format: string;
	    width: number;
	    height: number;
	    size: string;
	    frames: number;
	    thumbWidth: number;
	    thumbHeight: number;
	    thumbnail: string;
	    path: string;
	    noSuffixName: string;
	
	    static createFrom(source: any = {}) {
	        return new ImageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.format = source["format"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.size = source["size"];
	        this.frames = source["frames"];
	        this.thumbWidth = source["thumbWidth"];
	        this.thumbHeight = source["thumbHeight"];
	        this.thumbnail = source["thumbnail"];
	        this.path = source["path"];
	        this.noSuffixName = source["noSuffixName"];
	    }
	}
	export class ImageOptions {
	    op: number;
	    encoder: string;
	    outPath: string;
	    width: number;
	    height: number;
	    percent: number;
	    jpgQuality: number;
	    pngCompression: number;
	    gifNumColors: number;
	    gifDropRate: number;
	    gifDrawOnBefore: boolean;
	    webpLossless: boolean;
	    webpQuality: number;
	    webpRgbInTransparent: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImageOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.op = source["op"];
	        this.encoder = source["encoder"];
	        this.outPath = source["outPath"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.percent = source["percent"];
	        this.jpgQuality = source["jpgQuality"];
	        this.pngCompression = source["pngCompression"];
	        this.gifNumColors = source["gifNumColors"];
	        this.gifDropRate = source["gifDropRate"];
	        this.gifDrawOnBefore = source["gifDrawOnBefore"];
	        this.webpLossless = source["webpLossless"];
	        this.webpQuality = source["webpQuality"];
	        this.webpRgbInTransparent = source["webpRgbInTransparent"];
	    }
	}

}

