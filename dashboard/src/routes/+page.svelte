<script lang="ts">
	import AdminShell from '$lib/components/AdminShell.svelte';
	import { apiFetch } from '$lib/api';
	import type {
		Machine,
		PaginatedVisitLogGroups,
		PolicyConfig,
		PolicyRule,
		VisitLogGroup
	} from '$lib/types';
	import { onMount } from 'svelte';

	type AttentionMachine = {
		machine: Machine;
		blockedCount: number;
		reason: 'high_blocked_activity' | 'never_seen' | 'stale';
	};

	const cardClass =
		'rounded-2xl border border-black/10 bg-white/95 p-5 shadow-[0_16px_36px_rgba(0,0,0,0.05)]';
	const labelClass = 'text-xs font-bold tracking-[0.12em] text-[#555555] uppercase';
	const blockedActions = ['blocked_blacklist', 'blocked_default', 'blocked_title'];

	let machines = $state<Machine[]>([]);
	let rules = $state<PolicyRule[]>([]);
	let policyConfig = $state<PolicyConfig | null>(null);
	let recentBlocked = $state<VisitLogGroup[]>([]);
	let loading = $state(true);
	let errorMessage = $state('');

	const machineNameById = $derived(
		new Map(
			machines.map((machine) => [machine.machine_id, machine.display_name || machine.machine_id])
		)
	);
	const activeMachines = $derived(machines.filter((machine) => machine.status === 'active').length);
	const currentRuleAction = $derived(policyConfig?.default_action === 'block' ? 'allow' : 'block');
	const activeModeRules = $derived(
		rules.filter(
			(rule) =>
				rule.enabled &&
				rule.action === currentRuleAction &&
				rule.pattern_type !== 'title_contains_any'
		).length
	);
	const titleKeywordCount = $derived(
		rules
			.filter((rule) => rule.enabled && rule.pattern_type === 'title_contains_any')
			.flatMap((rule) => parseKeywordList(rule.pattern_value)).length
	);
	const blockedToday = $derived(
		recentBlocked
			.filter((log) => isToday(log.first_visited_at))
			.reduce((total, log) => total + log.visit_count, 0)
	);
	const machinesWithRecentBlocks = $derived(
		Array.from(new Set(recentBlocked.map((log) => log.machine_id))).length
	);
	const blockedCountByMachine = $derived(
		recentBlocked.reduce((counts, log) => {
			counts.set(log.machine_id, (counts.get(log.machine_id) || 0) + log.visit_count);
			return counts;
		}, new Map<string, number>())
	);
	const attentionMachines = $derived(
		machines
			.map((machine): AttentionMachine | null => {
				const blockedCount = blockedCountByMachine.get(machine.machine_id) || 0;
				if (blockedCount > 0) {
					return { machine, blockedCount, reason: 'high_blocked_activity' };
				}

				if (!machine.last_seen_at) {
					return { machine, blockedCount, reason: 'never_seen' };
				}

				if (Date.now() - new Date(machine.last_seen_at).getTime() > 24 * 60 * 60 * 1000) {
					return { machine, blockedCount, reason: 'stale' };
				}

				return null;
			})
			.filter((item): item is AttentionMachine => item !== null)
			.sort((left, right) => {
				if (right.blockedCount !== left.blockedCount) {
					return right.blockedCount - left.blockedCount;
				}

				return (
					new Date(right.machine.last_seen_at || 0).getTime() -
					new Date(left.machine.last_seen_at || 0).getTime()
				);
			})
	);

	function parseKeywordList(value: string) {
		try {
			const parsed = JSON.parse(value) as unknown;
			return Array.isArray(parsed)
				? parsed.filter((item): item is string => typeof item === 'string')
				: [];
		} catch {
			return [];
		}
	}

	function formatDate(value: string | null) {
		if (!value) return 'Never seen';
		return new Date(value).toLocaleString();
	}

	function isToday(value: string) {
		const date = new Date(value);
		const now = new Date();
		return (
			date.getFullYear() === now.getFullYear() &&
			date.getMonth() === now.getMonth() &&
			date.getDate() === now.getDate()
		);
	}

	function blockedReason(action: string) {
		if (action === 'blocked_title') return 'Title keyword';
		if (action === 'blocked_default') return 'Whitelist mode';
		return 'Blacklist rule';
	}

	function machineLabel(machineId: string) {
		return machineNameById.get(machineId) || machineId;
	}

	function attentionReasonLabel(reason: AttentionMachine['reason']) {
		if (reason === 'high_blocked_activity') return 'High blocked activity';
		if (reason === 'never_seen') return 'Never seen';
		return 'Stale machine';
	}

	function buildLogPath(action: string) {
		const query = new URLSearchParams({
			action,
			limit: '10',
			offset: '0'
		});
		return `/api/v1/admin/log-groups?${query.toString()}`;
	}

	onMount(async () => {
		try {
			const [machineItems, config, ruleItems, ...blockedResponses] = await Promise.all([
				apiFetch<Machine[]>('/api/v1/admin/machines'),
				apiFetch<PolicyConfig>('/api/v1/admin/policy-config'),
				apiFetch<PolicyRule[]>('/api/v1/admin/rules'),
				...blockedActions.map((action) => apiFetch<PaginatedVisitLogGroups>(buildLogPath(action)))
			]);

			machines = machineItems;
			policyConfig = config;
			rules = ruleItems;
			recentBlocked = blockedResponses
				.flatMap((response) => response.items)
				.sort(
					(left, right) =>
						new Date(right.last_visited_at).getTime() - new Date(left.last_visited_at).getTime()
				)
				.slice(0, 10);
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not load overview.';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>For Little Dashboard</title>
</svelte:head>

<AdminShell
	title="Overview"
	summary="Operational summary for shared machines, active policy, blocked activity, and machines needing attention."
>
	{#if loading}
		<div class={cardClass}>
			<p class="m-0 leading-relaxed text-[#555555]">Loading dashboard overview...</p>
		</div>
	{:else if errorMessage}
		<div class={cardClass}>
			<p class="m-0 leading-relaxed text-[#555555]">{errorMessage}</p>
		</div>
	{:else}
		<section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
			<article class={cardClass}>
				<p class={labelClass}>Machines</p>
				<strong class="mt-3 block text-3xl text-[#111111]">{machines.length}</strong>
				<p class="mt-2 text-[#555555]">{activeMachines} active</p>
			</article>

			<article class={cardClass}>
				<p class={labelClass}>Policy Mode</p>
				<strong class="mt-3 block text-2xl text-[#111111]">
					{policyConfig?.default_action === 'block' ? 'Whitelist Only' : 'Blacklist First'}
				</strong>
				<p class="mt-2 text-[#555555]">{activeModeRules} active domain rule(s)</p>
			</article>

			<article class={cardClass}>
				<p class={labelClass}>Title Keywords</p>
				<strong class="mt-3 block text-3xl text-[#111111]">{titleKeywordCount}</strong>
				<p class="mt-2 text-[#555555]">Blocked title keyword(s)</p>
			</article>

			<article class={cardClass}>
				<p class={labelClass}>Blocked Today</p>
				<strong class="mt-3 block text-3xl text-[#5f1818]">{blockedToday}</strong>
				<p class="mt-2 text-[#555555]">{machinesWithRecentBlocks} machine(s) recently blocked</p>
			</article>
		</section>

		<section class="mt-4 grid gap-4 xl:grid-cols-[1.4fr_1fr]">
			<article class={cardClass}>
				<div class="mb-4">
					<p class={labelClass}>Recent Blocked Activity</p>
					<h3 class="mt-2 mb-0 text-xl font-semibold text-[#111111]">Latest blocked visits</h3>
				</div>

				{#if recentBlocked.length === 0}
					<p class="m-0 leading-relaxed text-[#555555]">
						No blocked visits have been recorded yet.
					</p>
				{:else}
					<div class="grid gap-3">
						{#each recentBlocked as log (log.group_id)}
							<div class="rounded-2xl border border-black/10 bg-white p-4">
								<div class="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
									<div class="min-w-0">
										<p class="mb-1 text-xs font-bold tracking-[0.12em] text-[#3d3d3d] uppercase">
											{machineLabel(log.machine_id)}
										</p>
										<h4 class="m-0 text-base font-semibold [overflow-wrap:anywhere] text-[#111111]">
											{log.title || log.domain}
										</h4>
										<p class="mt-1 text-sm [overflow-wrap:anywhere] text-[#555555]">{log.domain}</p>
									</div>
									<span
										class="w-fit shrink-0 rounded-full bg-[#5f1818]/10 px-3 py-1.5 text-xs font-bold text-[#5f1818]"
									>
										{blockedReason(log.action)}
									</span>
								</div>
								<p class="mt-3 mb-0 text-sm text-[#555555]">
									{formatDate(log.last_visited_at)} · {log.visit_count} visit(s)
								</p>
							</div>
						{/each}
					</div>
				{/if}
			</article>

			<article class={cardClass}>
				<div class="mb-4">
					<p class={labelClass}>Machines Needing Attention</p>
					<h3 class="mt-2 mb-0 text-xl font-semibold text-[#111111]">Stale or pending machines</h3>
				</div>

				{#if attentionMachines.length === 0}
					<p class="m-0 leading-relaxed text-[#555555]">No machines need attention right now.</p>
				{:else}
					<div class="grid gap-3">
						{#each attentionMachines.slice(0, 8) as item (item.machine.id)}
							<div class="rounded-2xl border border-black/10 bg-white p-4">
								<h4 class="m-0 text-base font-semibold text-[#111111]">
									{item.machine.display_name || item.machine.machine_id}
								</h4>
								<p class="mt-1 text-sm text-[#555555]">
									Last seen: {formatDate(item.machine.last_seen_at)}
								</p>
								<div class="mt-3 flex flex-wrap gap-2">
									<span
										class={[
											'inline-flex rounded-full px-3 py-1.5 text-xs font-bold',
											item.reason === 'high_blocked_activity'
												? 'bg-[#5f1818]/10 text-[#5f1818]'
												: 'bg-black/[0.04] text-[#3d3d3d]'
										]}
									>
										{attentionReasonLabel(item.reason)}
									</span>
									{#if item.blockedCount > 0}
										<span
											class="inline-flex rounded-full bg-[#5f1818]/10 px-3 py-1.5 text-xs font-bold text-[#5f1818]"
										>
											{item.blockedCount} blocked visit(s)
										</span>
									{/if}
									<span
										class="inline-flex rounded-full bg-black/[0.04] px-3 py-1.5 text-xs font-bold text-[#3d3d3d]"
									>
										{item.machine.status}
									</span>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</article>
		</section>
	{/if}
</AdminShell>
