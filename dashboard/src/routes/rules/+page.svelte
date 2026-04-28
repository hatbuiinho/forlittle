<script lang="ts">
	import AdminShell from '$lib/components/AdminShell.svelte';
	import { apiFetch } from '$lib/api';
	import type { PolicyConfig, PolicyRule } from '$lib/types';
	import { onMount } from 'svelte';

	const buttonBaseClass =
		'inline-flex min-h-11 cursor-pointer items-center justify-center rounded-xl border px-4 py-2 font-bold disabled:cursor-not-allowed disabled:opacity-60';
	const buttonClass = `${buttonBaseClass} border-black/10 bg-white text-[#111111]`;
	const primaryButtonClass = `${buttonBaseClass} border-[#25441f] bg-[#25441f] text-white`;
	const dangerButtonClass = `${buttonBaseClass} border-[#5f1818]/20 bg-[#5f1818]/10 text-[#5f1818]`;
	const inputClass =
		'w-full min-w-0 rounded-xl border border-black/10 bg-white text-[#111111] focus:border-[#25441f] focus:ring-[#25441f]';
	const labelClass = 'grid min-w-0 gap-1.5';
	const labelTextClass = 'text-xs font-bold tracking-[0.08em] text-[#555555] uppercase';
	type PatternType = 'domain_suffix' | 'domain_exact' | 'domain_contains' | 'title_contains_any';

	let rules = $state<PolicyRule[]>([]);
	let policyConfig = $state<PolicyConfig | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let savingMode = $state(false);
	let errorMessage = $state('');
	let patternType = $state<PatternType>('domain_suffix');
	let patternValue = $state('');
	let editingRuleId = $state<number | null>(null);
	let editingPatternType = $state<PatternType>('domain_suffix');
	let editingPatternValue = $state('');
	let titleKeywordInput = $state('');
	let savingTitleKeywords = $state(false);

	const activeRuleAction = $derived(policyConfig?.default_action === 'block' ? 'allow' : 'block');
	const titleKeywordRule = $derived(
		rules.find((rule) => rule.action === 'block' && rule.pattern_type === 'title_contains_any') ||
			null
	);
	const titleKeywords = $derived(parseKeywordList(titleKeywordRule?.pattern_value || '[]'));
	const visibleRules = $derived(
		rules.filter(
			(rule) => rule.action === activeRuleAction && rule.pattern_type !== 'title_contains_any'
		)
	);
	const formActionLabel = $derived(activeRuleAction === 'allow' ? 'Allow' : 'Block');

	function parseKeywordList(value: string) {
		try {
			const parsed = JSON.parse(value) as unknown;
			if (!Array.isArray(parsed)) {
				return [];
			}

			return parsed.filter((item): item is string => typeof item === 'string');
		} catch {
			return [];
		}
	}

	function normalizeTitleKeyword(value: string) {
		return value.trim().toLowerCase().replace(/\s+/g, ' ');
	}

	function parseKeywordInput(value: string) {
		return Array.from(
			new Set(
				value
					.split(/[\n\r\t,;|]+/)
					.map(normalizeTitleKeyword)
					.filter(Boolean)
			)
		);
	}

	function parsePatternInput(value: string) {
		return Array.from(
			new Set(
				value
					.split(/[\n\r\t,;|]+/)
					.map((item) => item.trim())
					.filter(Boolean)
			)
		);
	}

	function normalizeContainsPattern(value: string) {
		const rawValue = value.trim().toLowerCase();
		if (!rawValue) {
			return { value: '', error: 'Domain keyword is required.' };
		}

		if (
			rawValue.includes('://') ||
			rawValue.includes('/') ||
			rawValue.includes('*') ||
			/\s/.test(rawValue)
		) {
			return { value: '', error: 'Use a plain keyword, for example edu.' };
		}

		if (!/^[a-z0-9.-]+$/.test(rawValue) || rawValue === '.' || rawValue === '-') {
			return { value: '', error: 'Domain keyword format is invalid.' };
		}

		return { value: rawValue, error: '' };
	}

	function normalizePattern(value: string, type: PatternType) {
		if (type === 'title_contains_any') {
			return { value: JSON.stringify(parseKeywordList(value)), error: '' };
		}

		if (type === 'domain_contains') {
			return normalizeContainsPattern(value);
		}

		const rawValue = value.trim().toLowerCase();
		if (!rawValue) {
			return { value: '', error: 'Domain pattern is required.' };
		}

		if (rawValue.includes('*') || /\s/.test(rawValue)) {
			return { value: '', error: 'Use a plain domain, for example facebook.com.' };
		}

		let domain = rawValue;
		if (domain.includes('://')) {
			try {
				domain = new URL(domain).hostname;
			} catch {
				return { value: '', error: 'Domain or URL is invalid.' };
			}
		} else {
			domain = domain.split('/')[0].split('?')[0].split('#')[0];
		}

		domain = domain.replace(/\.$/, '').replace(/^www\./, '');

		if (!domain || !domain.includes('.') || !/^[a-z0-9.-]+$/.test(domain)) {
			return { value: '', error: 'Use a valid domain, for example facebook.com.' };
		}

		if (domain.includes('..') || domain.startsWith('.') || domain.endsWith('-')) {
			return { value: '', error: 'Domain format is invalid.' };
		}

		return { value: domain, error: '' };
	}

	async function loadRules() {
		loading = true;
		errorMessage = '';

		try {
			const [config, ruleItems] = await Promise.all([
				apiFetch<PolicyConfig>('/api/v1/admin/policy-config'),
				apiFetch<PolicyRule[]>('/api/v1/admin/rules')
			]);
			policyConfig = config;
			rules = ruleItems;
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not load rules.';
		} finally {
			loading = false;
		}
	}

	async function updatePolicyMode(defaultAction: 'allow' | 'block') {
		if (policyConfig?.default_action === defaultAction) return;

		savingMode = true;
		errorMessage = '';
		cancelEditing();

		try {
			policyConfig = await apiFetch<PolicyConfig>('/api/v1/admin/policy-config', {
				method: 'PUT',
				body: JSON.stringify({ default_action: defaultAction })
			});
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not update policy mode.';
		} finally {
			savingMode = false;
		}
	}

	async function createRule() {
		const rawPatterns = parsePatternInput(patternValue);
		if (rawPatterns.length === 0) {
			errorMessage = 'Enter at least one domain pattern.';
			return;
		}

		const normalizedValues: string[] = [];
		for (const rawPattern of rawPatterns) {
			const normalized = normalizePattern(rawPattern, patternType);
			if (normalized.error) {
				errorMessage = `${rawPattern}: ${normalized.error}`;
				return;
			}
			normalizedValues.push(normalized.value);
		}

		const uniqueValues = Array.from(new Set(normalizedValues));
		if (uniqueValues.length === 0) {
			errorMessage = 'Enter at least one valid domain pattern.';
			return;
		}

		patternValue = uniqueValues.join('\n');
		saving = true;
		errorMessage = '';

		try {
			const createdRules = await Promise.all(
				uniqueValues.map((value) =>
					apiFetch<PolicyRule>('/api/v1/admin/rules', {
						method: 'POST',
						body: JSON.stringify({
							action: activeRuleAction,
							pattern_type: patternType,
							pattern_value: value,
							enabled: true
						})
					})
				)
			);

			rules = [...rules, ...createdRules];
			patternValue = '';
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not create rule.';
		} finally {
			saving = false;
		}
	}

	async function saveTitleKeywords(nextKeywords: string[]) {
		const normalizedKeywords = Array.from(
			new Set(nextKeywords.map(normalizeTitleKeyword).filter(Boolean))
		);

		savingTitleKeywords = true;
		errorMessage = '';

		try {
			if (titleKeywordRule) {
				const updatedRule = await apiFetch<PolicyRule>(
					`/api/v1/admin/rules/${titleKeywordRule.id}`,
					{
						method: 'PATCH',
						body: JSON.stringify({
							action: 'block',
							pattern_type: 'title_contains_any',
							pattern_value: JSON.stringify(normalizedKeywords),
							enabled: true
						})
					}
				);

				rules = rules.map((rule) => (rule.id === updatedRule.id ? updatedRule : rule));
				return;
			}

			const createdRule = await apiFetch<PolicyRule>('/api/v1/admin/rules', {
				method: 'POST',
				body: JSON.stringify({
					action: 'block',
					pattern_type: 'title_contains_any',
					pattern_value: JSON.stringify(normalizedKeywords),
					enabled: true
				})
			});

			rules = [...rules, createdRule];
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not update title keywords.';
		} finally {
			savingTitleKeywords = false;
		}
	}

	async function addTitleKeywords() {
		const nextKeywords = parseKeywordInput(titleKeywordInput);

		if (nextKeywords.length === 0) {
			errorMessage = 'Enter at least one title keyword.';
			return;
		}

		await saveTitleKeywords([...titleKeywords, ...nextKeywords]);
		titleKeywordInput = '';
	}

	async function removeTitleKeyword(keyword: string) {
		await saveTitleKeywords(titleKeywords.filter((item) => item !== keyword));
	}

	function startEditing(rule: PolicyRule) {
		editingRuleId = rule.id;
		editingPatternType = rule.pattern_type as PatternType;
		editingPatternValue = rule.pattern_value;
	}

	function cancelEditing() {
		editingRuleId = null;
		editingPatternValue = '';
	}

	async function saveRule(ruleId: number) {
		const normalized = normalizePattern(editingPatternValue, editingPatternType);
		if (normalized.error) {
			errorMessage = normalized.error;
			return;
		}

		editingPatternValue = normalized.value;
		errorMessage = '';

		try {
			const updatedRule = await apiFetch<PolicyRule>(`/api/v1/admin/rules/${ruleId}`, {
				method: 'PATCH',
				body: JSON.stringify({
					action: activeRuleAction,
					pattern_type: editingPatternType,
					pattern_value: normalized.value
				})
			});

			rules = rules.map((rule) => (rule.id === ruleId ? updatedRule : rule));
			cancelEditing();
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not update rule.';
		}
	}

	async function toggleRule(rule: PolicyRule) {
		errorMessage = '';

		try {
			const updatedRule = await apiFetch<PolicyRule>(`/api/v1/admin/rules/${rule.id}`, {
				method: 'PATCH',
				body: JSON.stringify({ enabled: !rule.enabled })
			});

			rules = rules.map((item) => (item.id === rule.id ? updatedRule : item));
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not update rule status.';
		}
	}

	async function deleteRule(rule: PolicyRule) {
		if (!confirm(`Delete rule for ${rule.pattern_value}?`)) {
			return;
		}

		errorMessage = '';

		try {
			await apiFetch<{ ok: boolean }>(`/api/v1/admin/rules/${rule.id}`, {
				method: 'DELETE'
			});

			rules = rules.filter((item) => item.id !== rule.id);
			if (editingRuleId === rule.id) {
				cancelEditing();
			}
		} catch (error) {
			errorMessage = error instanceof Error ? error.message : 'Could not delete rule.';
		}
	}

	function formatDate(value: string) {
		return new Date(value).toLocaleString();
	}

	function actionLabel(value: string) {
		return value === 'allow' ? 'Allow' : 'Block';
	}

	function patternTypeLabel(value: string) {
		if (value === 'domain_exact') return 'Exact domain';
		if (value === 'domain_contains') return 'Domain contains';
		return 'Domain and subdomains';
	}

	onMount(loadRules);
</script>

<svelte:head>
	<title>Rules | For Little</title>
</svelte:head>

<AdminShell
	title="Policy Rules"
	summary="Manage global allowlist and blocklist rules synced by every enrolled extension."
>
	<div
		class="mb-4 flex flex-col items-stretch justify-between gap-4 rounded-2xl border border-black/10 bg-white/95 p-4 shadow-[0_16px_36px_rgba(0,0,0,0.04)] md:flex-row md:items-center"
	>
		<div>
			<p class="mb-2 text-xs font-bold tracking-[0.14em] text-[#3d3d3d] uppercase">Policy Mode</p>
			<h3 class="m-0 text-lg font-semibold text-[#111111]">
				{policyConfig?.default_action === 'block' ? 'Whitelist Only' : 'Blacklist First'}
			</h3>
			<p class="mt-1 text-[#555555]">
				{policyConfig?.default_action === 'block'
					? 'Only allowed domains can open. Everything else is blocked.'
					: 'All domains can open unless a block rule matches.'}
			</p>
		</div>
		<div
			class="inline-flex gap-1 rounded-xl border border-black/10 bg-black/[0.04] p-1"
			aria-label="Policy mode"
		>
			<button
				type="button"
				class={[
					'cursor-pointer rounded-lg px-3 py-2 font-bold disabled:cursor-not-allowed disabled:opacity-70',
					policyConfig?.default_action === 'allow' ? 'bg-[#25441f] text-white' : 'text-[#555555]'
				]}
				disabled={savingMode || loading}
				onclick={() => updatePolicyMode('allow')}
			>
				Blacklist First
			</button>
			<button
				type="button"
				class={[
					'cursor-pointer rounded-lg px-3 py-2 font-bold disabled:cursor-not-allowed disabled:opacity-70',
					policyConfig?.default_action === 'block' ? 'bg-[#25441f] text-white' : 'text-[#555555]'
				]}
				disabled={savingMode || loading}
				onclick={() => updatePolicyMode('block')}
			>
				Whitelist Only
			</button>
		</div>
	</div>

	{#if policyConfig?.default_action === 'allow'}
		<section
			class="mb-4 rounded-2xl border border-[#5f1818]/15 bg-[#5f1818]/5 p-4 shadow-[0_16px_36px_rgba(0,0,0,0.04)]"
		>
			<div class="mb-4">
				<p class="mb-2 text-xs font-bold tracking-[0.14em] text-[#5f1818] uppercase">
					Blocked Title Keywords
				</p>
				<h3 class="m-0 text-lg font-semibold text-[#111111]">Block pages by title words</h3>
				<p class="mt-1 text-[#555555]">
					If a page title contains any keyword below, the extension blocks that tab after the title
					is detected.
				</p>
			</div>

			<div class="mb-4 flex flex-wrap gap-2">
				{#if titleKeywords.length === 0}
					<p class="m-0 text-sm text-[#555555]">No title keywords configured.</p>
				{:else}
					{#each titleKeywords as keyword (keyword)}
						<span
							class="inline-flex items-center gap-2 rounded-full border border-[#5f1818]/15 bg-white px-3 py-1.5 text-sm font-bold text-[#5f1818]"
						>
							{keyword}
							<button
								type="button"
								class="cursor-pointer border-0 bg-transparent p-0 font-bold text-[#5f1818]"
								disabled={savingTitleKeywords}
								aria-label={`Remove ${keyword}`}
								onclick={() => removeTitleKeyword(keyword)}
							>
								x
							</button>
						</span>
					{/each}
				{/if}
			</div>

			<div class="grid gap-3 md:grid-cols-[1fr_auto]">
				<label class={labelClass}>
					<span class={labelTextClass}>Add Keywords</span>
					<textarea
						class={inputClass}
						bind:value={titleKeywordInput}
						rows="3"
						placeholder="Paste from Excel, one keyword per row, or use comma-separated keywords"
					></textarea>
					<span class="text-sm leading-relaxed text-[#555555]">
						Accepted separators: new line, tab, comma, semicolon, or pipe.
					</span>
				</label>
				<button
					type="button"
					class={dangerButtonClass}
					disabled={savingTitleKeywords}
					onclick={addTitleKeywords}
				>
					{savingTitleKeywords ? 'Saving...' : 'Add Keywords'}
				</button>
			</div>
		</section>
	{/if}

	<form
		class="mb-4 grid gap-3 rounded-2xl border border-black/10 bg-white/95 p-4 md:grid-cols-[0.8fr_1fr] md:items-start"
		onsubmit={(event) => event.preventDefault()}
	>
		<div class={labelClass}>
			<span class={labelTextClass}>Rule Action</span>
			<span
				class={[
					'inline-flex min-h-11 items-center rounded-xl border px-3 font-bold',
					activeRuleAction === 'block'
						? 'border-[#5f1818]/20 bg-[#5f1818]/10 text-[#5f1818]'
						: 'border-[#25441f]/20 bg-[#496b3a]/15 text-[#25441f]'
				]}
			>
				{formActionLabel}
			</span>
		</div>

		<label class={labelClass}>
			<span class={labelTextClass}>Match Type</span>
			<select class={inputClass} bind:value={patternType}>
				<option value="domain_suffix">Domain and subdomains</option>
				<option value="domain_exact">Exact domain</option>
				<option value="domain_contains">Domain contains</option>
			</select>
		</label>

		<label class={`${labelClass} md:col-span-2`}>
			<span class={labelTextClass}>Domain Pattern</span>
			<textarea
				class={inputClass}
				bind:value={patternValue}
				rows="3"
				placeholder={patternType === 'domain_contains' ? 'edu' : 'facebook.com'}
			></textarea>
			<span class="text-sm leading-relaxed text-[#555555]">
				Paste from Excel or separate values by new line, tab, comma, semicolon, or pipe.
			</span>
		</label>

		<button
			type="button"
			class={`${primaryButtonClass} md:col-span-2 md:w-fit md:justify-self-end`}
			disabled={saving}
			onclick={createRule}
		>
			{saving ? 'Saving...' : 'Add Rule(s)'}
		</button>
	</form>

	{#if loading}
		<div class="rounded-2xl border border-black/10 bg-white/95 p-5">
			<p class="m-0 leading-relaxed text-[#555555]">Loading policy rules from the backend...</p>
		</div>
	{:else if errorMessage}
		<div class="rounded-2xl border border-black/10 bg-white/95 p-5">
			<p class="m-0 leading-relaxed text-[#555555]">{errorMessage}</p>
		</div>
	{/if}

	{#if !loading && visibleRules.length === 0}
		<div class="rounded-2xl border border-black/10 bg-white/95 p-5">
			<p class="m-0 leading-relaxed text-[#555555]">
				No {activeRuleAction === 'allow' ? 'allow' : 'block'} rules are active for this policy mode.
			</p>
		</div>
	{:else if visibleRules.length > 0}
		<div class="grid gap-3">
			{#each visibleRules as rule (rule.id)}
				<article
					class="flex flex-col gap-4 rounded-2xl border border-black/10 bg-white/95 p-4 shadow-[0_16px_36px_rgba(0,0,0,0.04)] md:flex-row md:flex-wrap md:items-start md:justify-between"
				>
					{#if editingRuleId === rule.id}
						<div class="grid w-full gap-3 md:grid-cols-[1fr_minmax(12rem,1.4fr)_auto] md:items-end">
							<label class={labelClass}>
								<span class={labelTextClass}>Match Type</span>
								<select class={inputClass} bind:value={editingPatternType}>
									<option value="domain_suffix">Domain and subdomains</option>
									<option value="domain_exact">Exact domain</option>
									<option value="domain_contains">Domain contains</option>
								</select>
							</label>
							<label class={labelClass}>
								<span class={labelTextClass}>Domain Pattern</span>
								<input
									class={inputClass}
									bind:value={editingPatternValue}
									type="text"
									placeholder={editingPatternType === 'domain_contains' ? 'edu' : 'facebook.com'}
								/>
							</label>
							<div class="flex flex-wrap gap-2">
								<button type="button" class={primaryButtonClass} onclick={() => saveRule(rule.id)}>
									Save
								</button>
								<button type="button" class={buttonClass} onclick={cancelEditing}>Cancel</button>
							</div>
						</div>
					{:else}
						<div>
							<p class="mb-2 text-xs font-bold tracking-[0.14em] text-[#3d3d3d] uppercase">
								{patternTypeLabel(rule.pattern_type)}
							</p>
							<h3 class="m-0 text-lg font-semibold text-[#111111]">{rule.pattern_value}</h3>
							<p class="mt-1 text-[#555555]">Created {formatDate(rule.created_at)}</p>
						</div>
						<div class="flex flex-wrap gap-2 md:justify-end">
							<span
								class={[
									'inline-flex items-center justify-center rounded-full px-3 py-1.5 text-xs font-bold',
									rule.action === 'block'
										? 'bg-[#5f1818]/10 text-[#5f1818]'
										: 'bg-[#496b3a]/15 text-[#25441f]'
								]}
							>
								{actionLabel(rule.action)}
							</span>
							<span
								class={[
									'inline-flex items-center justify-center rounded-full px-3 py-1.5 text-xs font-bold',
									rule.enabled ? 'bg-[#496b3a]/15 text-[#25441f]' : 'bg-[#496b3a]/10 text-[#25441f]'
								]}
							>
								{rule.enabled ? 'Enabled' : 'Disabled'}
							</span>
						</div>
						<div class="flex flex-wrap gap-2 md:justify-end">
							<button type="button" class={buttonClass} onclick={() => startEditing(rule)}
								>Edit</button
							>
							<button type="button" class={buttonClass} onclick={() => toggleRule(rule)}>
								{rule.enabled ? 'Disable' : 'Enable'}
							</button>
							<button type="button" class={dangerButtonClass} onclick={() => deleteRule(rule)}>
								Delete
							</button>
						</div>
					{/if}
				</article>
			{/each}
		</div>
	{/if}
</AdminShell>
