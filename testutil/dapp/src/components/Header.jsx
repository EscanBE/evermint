import { useLocation } from 'preact-iso';

export function Header() {
	const { url } = useLocation();

	return (
		<header>
			<nav>
				<a href="/" className={url === '/' && 'active'}>
					Home
				</a>
				<a href="/404" className={url === '/404' && 'active'}>
					404
				</a>
			</nav>
		</header>
	);
}
