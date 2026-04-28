export type LittleMonk = {
	id: number;
	code: string;
	display_name: string;
	status: string;
	created_at: string;
};

export type User = {
	id: number;
	email: string;
	display_name: string;
	role: string;
	status: string;
};

export type AuthResponse = {
	user: User;
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
	little_monk_id: number | null;
	action: string;
	pattern_type: string;
	pattern_value: string;
	enabled: boolean;
	created_at: string;
};

export type PolicyConfig = {
	id: number;
	default_action: 'allow' | 'block';
	created_at: string;
	updated_at: string;
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

export type PaginatedVisitLogs = {
	items: VisitLog[];
	total: number;
	limit: number;
	offset: number;
};

export type VisitLogGroup = {
	group_id: string;
	machine_id: string;
	profile_instance_id: string;
	domain: string;
	url: string;
	title: string;
	action: string;
	visit_count: number;
	first_visited_at: string;
	last_visited_at: string;
};

export type PaginatedVisitLogGroups = {
	items: VisitLogGroup[];
	total: number;
	limit: number;
	offset: number;
};
