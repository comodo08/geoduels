import type { GetServerSideProps } from 'next';
import { getSiteURL } from '../lib/site';

export const getServerSideProps: GetServerSideProps = async ({ res }) => {
  const siteURL = getSiteURL();

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
