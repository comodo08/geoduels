import { readServerConfig } from '../lib/runtime-config.server';
import type { GetServerSideProps } from 'next';

export const getServerSideProps: GetServerSideProps = async ({ res }) => {
  const siteURL = readServerConfig().siteURL;

  res.setHeader('Content-Type', 'text/plain');
  res.write(`User-agent: *\nAllow: /\n\nSitemap: ${siteURL}/sitemap.xml\n`);
  res.end();

  return {
    props: {}
  };
};

export default function RobotsTXT() {
  return null;
}
