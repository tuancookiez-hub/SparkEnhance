export namespace main {
	
	export class EnhanceInput {
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new EnhanceInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	    }
	}
	export class EnhanceResult {
	    output: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new EnhanceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output = source["output"];
	        this.score = source["score"];
	    }
	}
	export class SetupInput {
	    apiKey: string;
	    baseUrl: string;
	    model: string;
	    hotkey: string;
	    autoPaste: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SetupInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.baseUrl = source["baseUrl"];
	        this.model = source["model"];
	        this.hotkey = source["hotkey"];
	        this.autoPaste = source["autoPaste"];
	    }
	}

}

