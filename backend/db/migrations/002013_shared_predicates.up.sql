-- Shared predicates that were copy-pasted across many queries. Named functions
-- make the semantics explicit and keep the definition in one place.

CREATE FUNCTION public.gd_is_admin(p_user_id uuid) RETURNS boolean
LANGUAGE sql STABLE PARALLEL SAFE AS
$$ SELECT EXISTS (SELECT 1 FROM public.user_roles r WHERE r.user_id = p_user_id AND r.role = 'admin') $$;

CREATE FUNCTION public.gd_is_judge(p_user_id uuid) RETURNS boolean
LANGUAGE sql STABLE PARALLEL SAFE AS
$$ SELECT EXISTS (SELECT 1 FROM public.user_roles r WHERE r.user_id = p_user_id AND r.role = 'judge') $$;

CREATE FUNCTION public.gd_is_banned(p_user_id uuid) RETURNS boolean
LANGUAGE sql STABLE PARALLEL SAFE AS
$$ SELECT EXISTS (
     SELECT 1 FROM public.users u
     WHERE u.id = p_user_id
       AND u.banned_at IS NOT NULL
       AND (u.ban_expires_at IS NULL OR u.ban_expires_at > now())
   ) $$;

CREATE FUNCTION public.gd_display_name(p_display_name text, p_provider_name text, p_user_id uuid) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS
$$ SELECT coalesce(nullif(p_display_name, ''), nullif(p_provider_name, ''), p_user_id::text) $$;
