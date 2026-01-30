export namespace app {
	
	export class Stats {
	    totalRecords: number;
	    levelCounts: Record<string, number>;
	    lastUpdated: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalRecords = source["totalRecords"];
	        this.levelCounts = source["levelCounts"];
	        this.lastUpdated = source["lastUpdated"];
	    }
	}

}

export namespace domain {
	
	export class Aggregation {
	    function: string;
	    field?: string;
	    alias?: string;
	
	    static createFrom(source: any = {}) {
	        return new Aggregation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.function = source["function"];
	        this.field = source["field"];
	        this.alias = source["alias"];
	    }
	}
	export class FilterCondition {
	    type: string;
	    field: string;
	    value: any;
	    operator?: string;
	
	    static createFrom(source: any = {}) {
	        return new FilterCondition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.field = source["field"];
	        this.value = source["value"];
	        this.operator = source["operator"];
	    }
	}
	export class ImportResult {
	    totalRecords: number;
	    processed: number;
	    errors?: string[];
	    duration: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalRecords = source["totalRecords"];
	        this.processed = source["processed"];
	        this.errors = source["errors"];
	        this.duration = source["duration"];
	    }
	}
	export class ParserConfig {
	    type: string;
	    pattern?: string;
	    fields?: Record<string, string>;
	    timeFormat?: string;
	
	    static createFrom(source: any = {}) {
	        return new ParserConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.pattern = source["pattern"];
	        this.fields = source["fields"];
	        this.timeFormat = source["timeFormat"];
	    }
	}
	export class Query {
	    filters: FilterCondition[];
	    groupBy?: string[];
	    aggregations?: Aggregation[];
	    sortBy?: string;
	    sortDesc?: boolean;
	    limit?: number;
	    offset?: number;
	
	    static createFrom(source: any = {}) {
	        return new Query(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filters = this.convertValues(source["filters"], FilterCondition);
	        this.groupBy = source["groupBy"];
	        this.aggregations = this.convertValues(source["aggregations"], Aggregation);
	        this.sortBy = source["sortBy"];
	        this.sortDesc = source["sortDesc"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
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
	export class Record {
	    id: string;
	    timestamp: number;
	    level: string;
	    message: string;
	    service?: string;
	    fields?: Record<string, any>;
	    raw: string;
	
	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = source["timestamp"];
	        this.level = source["level"];
	        this.message = source["message"];
	        this.service = source["service"];
	        this.fields = source["fields"];
	        this.raw = source["raw"];
	    }
	}
	export class QueryResult {
	    records: Record[];
	    aggregations?: Record<string, any>;
	    total: number;
	    took: number;
	
	    static createFrom(source: any = {}) {
	        return new QueryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.records = this.convertValues(source["records"], Record);
	        this.aggregations = source["aggregations"];
	        this.total = source["total"];
	        this.took = source["took"];
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
	
	export class TimelinePoint {
	    bucketStart: number;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new TimelinePoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bucketStart = source["bucketStart"];
	        this.count = source["count"];
	    }
	}
	export class TimelineRequest {
	    filters: FilterCondition[];
	    bucketMs: number;
	
	    static createFrom(source: any = {}) {
	        return new TimelineRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filters = this.convertValues(source["filters"], FilterCondition);
	        this.bucketMs = source["bucketMs"];
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

}

