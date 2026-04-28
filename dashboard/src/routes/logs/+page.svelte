<script lang="ts">
	import AdminShell from '$lib/components/AdminShell.svelte';
	import { apiFetch } from '$lib/api';
	import type { Machine, PaginatedVisitLogGroups, VisitLogGroup } from '$lib/types';
	import { onMount } from 'svelte';

	const actions = [
		{ value: '', label: 'All actions' },
		{ value: 'allowed', label: 'Allowed' },
		{ value: 'allowed_whitelist', label: 'Allowed by whitelist' },
		{ value: 'blocked_blacklist', label: 'Blocked by blacklist' },
		{ value: 'blocked_default', label: 'Blocked by whitelist mode' },
		{ value: 'blocked_title', label: 'Blocked by title keyword' }
	];

	const buttonBaseClass =
		'inline-flex min-h-11 cursor-pointer items-center justify-center rounded-xl border px-4 py-2 font-bold disabled:cursor-not-allowed disabled:opacity-60';
	const buttonClass = `${buttonBaseClass} border-black/10 bg-white text-[#111111]`;
	const primaryButtonClass = `${buttonBaseClass} border-[#25441f] bg-[#25441f] text-white`;
	const inputClass =
		'w-full min-w-0 rounded-xl border border-black/10 bg-white text-[#111111] focus:border-[#25441f] focus:ring-[#25441f]';
	const labelClass = 'grid min-w-0 gap-1.5';
	const labelTextClass = 'text-xs font-bold tracking-[0.08em] text-[#555555] uppercase';

	let logGroups = $state<VisitLogGroup[]>([]);
	let machines = $state<Machine[]>([]);
	let loading = $state(true);
	let errorMessage = $state('');
	let machineId = $state('');
	let search = $state('');
	let action = $state('');
	let from = $state('');
	let to = $state('');
	let page = $state(1);
	let pageSize = $state(25);
	let totalLogs = $state(0);
	let isFilterOpen = $state(false);

	const machineNameById = $derived(
		new Map(
			machines.map((machine) => [machine.machine_id, machine.display_name || machine.machine_id])
		)
	);
	const totalPages = $derived(Math.max(1, Math.ceil(totalLogs / pageSize)));
	const pageStart = $derived(totalLogs === 0 ? 0 : (page - 1) * pageSize + 1);
	const pageEnd = $derived(Math.min(totalLogs, (page - 1) * pageSize + logGroups.length));
	const activeFilterCount = $derived(
		[machineId, search.trim(), action, from, to].filter(Boolean).length
	);

	function formatDate(value: string) {
		return new Date(value).toLocaleString();
	}

	function machineLabel(log: VisitLogGroup) {
		return machineNameById.get(log.machine_id) || log.machine_id;
	}

	function formatRange(log: VisitLogGroup) {
		if (log.first_visited_at === log.last_visited_at) {
			return formatDate(log.first_visited_at);
		}

		return `${formatDate(log.first_visited_at)} - ${formatDate(log.last_visited_at)}`;
	}

	function actionLabel(value: string) {
		switch (value) {
			case 'allowed_whitelist':
				return 'Allowed by whitelist';
			case 'blocked_blacklist':
				return 'Blocked by blacklist';
			case 'blocked_default':
				return 'Blocked by whitelist mode';
			case 'blocked_title':
				return 'Blocked by title keyword';
			default:
				return 'Allowed';
		}
	}

	function buildLogsPath() {
		const params = [
			machineId ? ['machine_id', machineId] : null,
			search.trim() ? ['search', search.trim()] : null,
			action ? ['action', action] : null,
			from ? ['from', new Date(from).toISOString()] : null,
			to ? ['to', new Date(to).toISOString()] : null,
			['limit', String(pageSize)],
			['offset', String((page - 1) * pageSize)]
		].filter((item): item is string[] => item !== null);

		const query = params
			.map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
			.join('&');
		return `/api/v1/admin/log-groups${query ? `?${query}` : ''}`;
	}

	async function loadLogs() {
		loading = true;
		errorMessage = '';

		try {
			const response = await apiFetch<PaginatedVisitLogGroups>(buildLogsPath());
			logGroups = response.items;
			totalLogs = response.total;
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not load visit logs.';
		} finally {
			loading = false;
		}
	}

	function applyFilters() {
		page = 1;
		isFilterOpen = false;
		void loadLogs();
	}

	function clearFilters() {
		machineId = '';
		search = '';
		action = '';
		from = '';
		to = '';
		page = 1;
		isFilterOpen = false;
		void loadLogs();
	}

	function changePage(nextPage: number) {
		page = Math.min(Math.max(nextPage, 1), totalPages);
		void loadLogs();
	}

	onMount(async () => {
		try {
			const [machineItems, logResponse] = await Promise.all([
				apiFetch<Machine[]>('/api/v1/admin/machines'),
				apiFetch<PaginatedVisitLogGroups>(buildLogsPath())
			]);
			machines = machineItems;
			logGroups = logResponse.items;
			totalLogs = logResponse.total;
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not load visit logs.';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Logs | For Little</title>
</svelte:head>

<AdminShell
	title="Visit Logs"
	summary="Review which machine opened which page, when it happened, and whether the visit was allowed or blocked."
>
	<div
		class="sticky top-0 z-10 mb-4 rounded-2xl border border-black/10 bg-white/95 p-3 shadow-[0_16px_30px_rgba(0,0,0,0.06)] backdrop-blur md:p-4"
	>
		<div class="flex items-center justify-between gap-3 md:hidden">
			<div>
				<p class="m-0 text-xs font-bold tracking-[0.14em] text-[#3d3d3d] uppercase">Filters</p>
				<p class="m-0 text-sm text-[#555555]">
					{activeFilterCount ? `${activeFilterCount} active` : 'Showing all visit groups'}
				</p>
			</div>
			<button
				type="button"
				class={buttonClass}
				aria-expanded={isFilterOpen}
				onclick={() => (isFilterOpen = !isFilterOpen)}
			>
				{isFilterOpen ? 'Hide Filters' : 'Show Filters'}
			</button>
		</div>

		<div
			class={[
				'grid-cols-1 gap-3 border-black/10 md:grid md:grid-cols-2 md:border-0 md:pt-0 xl:grid-cols-3',
				isFilterOpen ? 'mt-3 grid border-t pt-3' : 'hidden md:grid'
			]}
		>
			<label class={labelClass}>
				<span class={labelTextClass}>Machine</span>
				<select class={inputClass} bind:value={machineId}>
					<option value="">All machines</option>
					{#each machines as machine (machine.id)}
						<option value={machine.machine_id}>{machine.display_name || machine.machine_id}</option>
					{/each}
				</select>
			</label>

			<label class={labelClass}>
				<span class={labelTextClass}>Search</span>
				<input
					class={inputClass}
					bind:value={search}
					type="text"
					placeholder="Domain or page title"
				/>
			</label>

			<label class={labelClass}>
				<span class={labelTextClass}>Action</span>
				<select class={inputClass} bind:value={action}>
					{#each actions as item (item.value)}
						<option value={item.value}>{item.label}</option>
					{/each}
				</select>
			</label>

			<label class={labelClass}>
				<span class={labelTextClass}>From</span>
				<input class={inputClass} bind:value={from} type="datetime-local" />
			</label>

			<label class={labelClass}>
				<span class={labelTextClass}>To</span>
				<input class={inputClass} bind:value={to} type="datetime-local" />
			</label>

			<label class={labelClass}>
				<span class={labelTextClass}>Per Page</span>
				<select class={inputClass} bind:value={pageSize} onchange={applyFilters}>
					<option value={10}>10</option>
					<option value={25}>25</option>
					<option value={50}>50</option>
					<option value={100}>100</option>
				</select>
			</label>

			<div class="flex gap-2 md:col-span-2 md:justify-end xl:col-span-3">
				<button type="button" class={primaryButtonClass} onclick={applyFilters}>Apply</button>
				<button type="button" class={buttonClass} onclick={clearFilters}>Clear</button>
			</div>
		</div>
	</div>

	{#if loading}
		<div class="rounded-2xl border border-black/10 bg-white/95 p-5">
			<p class="m-0 leading-relaxed text-[#555555]">Loading visit logs from the backend...</p>
		</div>
	{:else if errorMessage}
		<div class="rounded-2xl border border-black/10 bg-white/95 p-5">
			<p class="m-0 leading-relaxed text-[#555555]">{errorMessage}</p>
		</div>
	{:else if logGroups.length === 0}
		<div class="rounded-2xl border border-black/10 bg-white/95 p-5">
			<p class="m-0 leading-relaxed text-[#555555]">No visit groups match the current filters.</p>
		</div>
	{:else}
		<div class="grid gap-3">
			{#each logGroups as log (log.group_id)}
				<article
					class="min-w-0 overflow-hidden rounded-2xl border border-black/10 bg-white/95 p-4 shadow-[0_16px_36px_rgba(0,0,0,0.04)]"
				>
					<div
						class="flex min-w-0 flex-col items-stretch gap-3 md:flex-row md:items-start md:justify-between"
					>
						<div class="min-w-0 flex-1">
							<p class="mb-2 text-xs font-bold tracking-[0.14em] text-[#3d3d3d] uppercase">
								{machineLabel(log)}
							</p>
							<h3 class="m-0 text-lg font-semibold [overflow-wrap:anywhere] text-[#111111]">
								{log.title || log.domain}
							</h3>
							<p class="mt-1 [overflow-wrap:anywhere] text-[#555555]">{log.url}</p>
						</div>
						<span
							class={[
								'inline-flex w-fit shrink-0 items-center justify-center rounded-full px-3 py-1.5 text-xs font-bold',
								log.action === 'blocked_blacklist' ||
								log.action === 'blocked_default' ||
								log.action === 'blocked_title'
									? 'bg-[#5f1818]/10 text-[#5f1818]'
									: 'bg-[#496b3a]/15 text-[#25441f]'
							]}
						>
							{actionLabel(log.action)}
						</span>
					</div>
					<dl
						class="mt-4 grid min-w-0 gap-3 md:grid-cols-[minmax(0,0.7fr)_minmax(0,1fr)_minmax(0,1.6fr)]"
					>
						<div class="min-w-0">
							<dt class="text-xs font-bold tracking-[0.08em] text-[#555555] uppercase">Visits</dt>
							<dd class="mt-1 text-[#111111]">{log.visit_count}</dd>
						</div>
						<div class="min-w-0">
							<dt class="text-xs font-bold tracking-[0.08em] text-[#555555] uppercase">Website</dt>
							<dd class="mt-1 [overflow-wrap:anywhere] text-[#111111]">{log.domain}</dd>
						</div>
						<div class="min-w-0">
							<dt class="text-xs font-bold tracking-[0.08em] text-[#555555] uppercase">When</dt>
							<dd class="mt-1 [overflow-wrap:anywhere] text-[#111111]">{formatRange(log)}</dd>
						</div>
					</dl>
				</article>
			{/each}
		</div>

		<div
			class="mt-4 flex flex-col items-stretch justify-between gap-4 rounded-2xl border border-black/10 bg-white/95 p-4 md:flex-row md:items-center"
		>
			<p class="m-0 text-[#555555]">Showing {pageStart}-{pageEnd} of {totalLogs} groups</p>
			<div class="flex items-center justify-between gap-3">
				<button
					type="button"
					class={buttonClass}
					disabled={page <= 1 || loading}
					onclick={() => changePage(page - 1)}
				>
					Previous
				</button>
				<span class="font-bold text-[#111111]">Page {page} of {totalPages}</span>
				<button
					type="button"
					class={buttonClass}
					disabled={page >= totalPages || loading}
					onclick={() => changePage(page + 1)}
				>
					Next
				</button>
			</div>
		</div>
	{/if}
</AdminShell>
