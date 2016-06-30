
#[macro_use]
extern crate mime;
extern crate iron;
extern crate html;
extern crate router;
extern crate postgres;
extern crate r2d2;
extern crate r2d2_postgres;

use iron::prelude::*;
use iron::status;
use router::Router;
use html::{html, head, title, text, text_str, body, h1, h2, ul, li, a};
use std::iter::FromIterator;

fn html_response(element: html::Element) -> IronResult<Response> {
    let mut content = String::from("<!DOCTYPE html>");

    content.push_str(&element.to_string());

    Ok(Response::with((
        mime!(Text/Html; Charset=Utf8),
        status::Ok,
        content
    )))
}

fn main() {
    let db_url = "postgres://postgres:postgres@localhost/dabbalist";
    let db_mgr = r2d2_postgres::PostgresConnectionManager::new(
        db_url,
        r2d2_postgres::SslMode::None
    ).unwrap();
    let db_pool = r2d2::Pool::new(r2d2::Config::default(), db_mgr).unwrap();

    let mut router = Router::new();

    router.get("/", move |_: &mut Request| -> IronResult<Response> {
        let conn = db_pool.get().unwrap();

        let book_list = Vec::from_iter(conn.query(" \
            SELECT id, title, author, translator FROM book \
        ", &[]).unwrap().into_iter().map(|row| {
            let id: i32 = row.get(0);
            let title: String = row.get(1);
            let author: String = row.get(2);
            let translator: String = row.get(3);

            let link_text = if translator == "" {
                format!("{}, by {}", &title, &author)
            } else {
                format!("{}, by {}, translation by {}", &title, &author, &translator)
            };

            let link = format!("/book/{}", id);

            li(vec![], vec![
                a(vec![("href", link)], vec![text(link_text)])
            ])
        }));

        html_response(html(vec![], vec![
            head(vec![], vec![
                title(vec![], vec![text_str("The Dabbalist")]),
            ]),
            body(vec![], vec![
                h1(vec![], vec![text_str("The Dabbalist")]),
                h2(vec![], vec![text_str("Books")]),
                ul(vec![], book_list)
            ])
        ]))
    });

    Iron::new(router).http("localhost:3000").unwrap();
    println!("Listening on 3000");
}

