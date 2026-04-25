export type LittleMonk = {
	id: number;
	code: string;
	display_name: string;
	status: string;
	created_at: string;
};

export type Machine = {
	id: number;
	machine_id: string;
	display_name: string;
	status: string;
	little_monk_id: number | null;
	last_seen_at: string | null;
	created_at: string;
};

export type PolicyRule = {
	id: number;
	little_monk_id: number;
	action: string;
	pattern_type: string;
	pattern_value: string;
	enabled: boolean;
	created_at: string;
};

export type VisitLog = {
	id: number;
	machine_id: string;
	profile_instance_id: string;
	tab_id: number;
	url: string;
	domain: string;
	title: string;
	visited_at: string;
	action: string;
};
