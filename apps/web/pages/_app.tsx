import type { AppProps } from 'next/app';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Head from 'next/head';
import { Montserrat } from 'next/font/google';
import { useState } from 'react';
import 'leaflet/dist/leaflet.css';
import 'easymde/dist/easymde.min.css';
import '../styles/globals.css';

const montserrat = Montserrat({
  subsets: ['latin'],
  variable: '--font-montserrat'
});

export default function App({ Component, pageProps }: AppProps) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            refetchOnWindowFocus: false,
            retry: false
          }
        }
      })
  );

  return (
    <>
      <Head>
        <link rel="icon" href="/icon.v1.png" type="image/png" />
        <link rel="shortcut icon" href="/icon.v1.png" type="image/png" />
        <link rel="apple-touch-icon" href="/icon.v1.png" />
      </Head>
      <div className={montserrat.variable}>
        <QueryClientProvider client={queryClient}>
          <Component {...pageProps} />
        </QueryClientProvider>
      </div>
    </>
  );
}
